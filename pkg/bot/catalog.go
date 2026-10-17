package bot

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed catalog.json
var embeddedCatalogJSON []byte

type CatalogSpecies struct {
	ID        int      `json:"id"`
	Slug      string   `json:"slug"`
	Name      string   `json:"name"`
	Types     []string `json:"types"`
	HuntLevel int      `json:"huntLevel"`
	Rarity    string   `json:"rarity"`
}

type HuntSpeciesRef struct {
	ID          int `json:"id"`
	GoldPerKill int `json:"goldPerKill"`
	XPPerKill   int `json:"xpPerKill"`
}

type HuntCard struct {
	Species          []HuntSpeciesRef `json:"species"`
	GoldPerKill      int              `json:"goldPerKill"`
	XPPerKill        int              `json:"xpPerKill"`
	Population       int              `json:"population"`
	DamageMultiplier float64          `json:"damageMultiplier"`
}

type CatalogHunt struct {
	Slug            string    `json:"slug"`
	Name            string    `json:"name"`
	MinTrainerLevel int       `json:"minTrainerLevel"`
	Region          string    `json:"region"`
	Card            *HuntCard `json:"card,omitempty"`
}

type CatalogBall struct {
	ID       int    `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Strength int    `json:"strength"`
}

type CatalogData struct {
	Species []CatalogSpecies `json:"species"`
	Hunts   []CatalogHunt   `json:"hunts"`
	Balls   []CatalogBall   `json:"balls"`
}

type Catalog struct {
	data          CatalogData
	speciesByID   map[int]CatalogSpecies
	speciesByName map[string]CatalogSpecies
	huntsBySlug   map[string]CatalogHunt
	ballsByID     map[int]CatalogBall
	mu            sync.RWMutex
}

var (
	globalCatalog     *Catalog
	globalCatalogOnce sync.Once
)

// GetGlobalCatalog retorna a instância única do catálogo carregado
func GetGlobalCatalog() *Catalog {
	globalCatalogOnce.Do(func() {
		cat := &Catalog{
			speciesByID:   make(map[int]CatalogSpecies),
			speciesByName: make(map[string]CatalogSpecies),
			huntsBySlug:   make(map[string]CatalogHunt),
			ballsByID:     make(map[int]CatalogBall),
		}

		if len(embeddedCatalogJSON) > 0 {
			_ = json.Unmarshal(embeddedCatalogJSON, &cat.data)
			for _, s := range cat.data.Species {
				cat.speciesByID[s.ID] = s
				cat.speciesByName[strings.ToLower(s.Name)] = s
			}
			for _, h := range cat.data.Hunts {
				cat.huntsBySlug[strings.ToLower(h.Slug)] = h
			}
			for _, b := range cat.data.Balls {
				cat.ballsByID[b.ID] = b
			}
		}

		globalCatalog = cat
	})
	return globalCatalog
}

func (c *Catalog) GetSpecies(id int) (CatalogSpecies, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.speciesByID[id]
	return s, ok
}

func (c *Catalog) GetSpeciesName(id int) string {
	if s, ok := c.GetSpecies(id); ok {
		return s.Name
	}
	return "Pokémon Desconhecido"
}

func (c *Catalog) GetHunt(slug string) (CatalogHunt, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	h, ok := c.huntsBySlug[strings.ToLower(slug)]
	return h, ok
}

func (c *Catalog) GetHuntName(slug string) string {
	if h, ok := c.GetHunt(slug); ok {
		return h.Name
	}
	if slug == "" {
		return "Nenhuma"
	}
	return slug
}

func (c *Catalog) GetAllHunts() []CatalogHunt {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]CatalogHunt, len(c.data.Hunts))
	copy(res, c.data.Hunts)
	return res
}

func (c *Catalog) GetBall(id int) (CatalogBall, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	b, ok := c.ballsByID[id]
	return b, ok
}

var unroutableHunts = map[string]bool{
	"vaporeon": true,
	"flareon":  true,
	"jolteon":  true,
}

// GetOptimalHunt retorna a rota com maior taxa de XP/Gold permitida para o nível do treinador
func (c *Catalog) GetOptimalHunt(level int) string {
	switch {
	case level >= 100:
		return "electabuzz"
	case level >= 80:
		return "omastar"
	case level >= 60:
		return "chansey"
	case level >= 50:
		return "tangela"
	case level >= 40:
		return "golbat"
	case level >= 30:
		return "drowzee"
	case level >= 20:
		return "staryu"
	case level >= 10:
		return "horsea"
	default:
		return "poliwag"
	}
}

// GetOptimalHuntForLeader seleciona a rota ideal combinando taxa de XP e vantagem de tipo elemental
func (c *Catalog) GetOptimalHuntForLeader(level int, leaderSpeciesID int) (string, string) {
	fallbackSlug := c.GetOptimalHunt(level)
	if leaderSpeciesID <= 0 {
		return fallbackSlug, "padrão de nível (sem líder)"
	}

	leader, hasLeader := c.GetSpecies(leaderSpeciesID)
	if !hasLeader || len(leader.Types) == 0 {
		return fallbackSlug, "padrão de nível (líder sem tipo)"
	}

	tiers := []int{280, 260, 240, 220, 200, 100, 80, 60, 50, 40, 30, 20, 10, 1}
	var maxTier int
	for _, t := range tiers {
		if level >= t {
			maxTier = t
			break
		}
	}
	if maxTier <= 0 {
		maxTier = 1
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	var candidateHunts []CatalogHunt
	for _, h := range c.data.Hunts {
		if h.MinTrainerLevel == maxTier && !unroutableHunts[h.Slug] {
			candidateHunts = append(candidateHunts, h)
		}
	}

	if len(candidateHunts) == 0 {
		return fallbackSlug, "padrão de nível (sem rotas no tier)"
	}

	type ScoredHunt struct {
		slug       string
		name       string
		score      float64
		offMult    float64
		defRisk    float64
		xp         int
		targetName string
	}

	var best ScoredHunt

	for _, h := range candidateHunts {
		if h.Card == nil || len(h.Card.Species) == 0 {
			continue
		}

		xp := h.Card.XPPerKill
		if xp <= 0 {
			xp = 1
		}

		totalOff := 0.0
		totalRisk := 0.0
		validSpecies := 0
		primaryTargetName := ""

		for _, sRef := range h.Card.Species {
			wild, ok := c.speciesByID[sRef.ID]
			if !ok {
				continue
			}
			if primaryTargetName == "" {
				primaryTargetName = wild.Name
			}
			off := GetBestOffensiveMultiplier(leader.Types, wild.Types)
			risk := GetIncomingRisk(leader.Types, wild.Types)

			totalOff += off
			totalRisk += risk
			validSpecies++
		}

		if validSpecies == 0 {
			continue
		}

		avgOff := totalOff / float64(validSpecies)
		avgRisk := totalRisk / float64(validSpecies)
		if avgRisk <= 0.0 {
			avgRisk = 0.25
		}

		// Se o golpe é completamente imune (0.0x), descarta a rota
		if avgOff <= 0.0 {
			continue
		}

		dmgMult := h.Card.DamageMultiplier
		if dmgMult <= 0.0 {
			dmgMult = 1.0
		}

		offFactor := avgOff
		if avgOff < 1.0 {
			offFactor = avgOff * 0.25 // Pesada penalidade por causar dano reduzido (lento)
		} else if avgOff >= 2.0 {
			offFactor = avgOff * 1.5 // Bônus por matar super rápido
		}

		effectiveRisk := avgRisk * (0.6 + 0.4*dmgMult)

		// Fórmula tática equilibrada: Prioriza XP real e rapidez de abates, evitando desvantagens ofensivas
		tacticalScore := (float64(xp) * offFactor) / effectiveRisk

		if tacticalScore > best.score {
			best = ScoredHunt{
				slug:       h.Slug,
				name:       h.Name,
				score:      tacticalScore,
				offMult:    avgOff,
				defRisk:    avgRisk,
				xp:         xp,
				targetName: primaryTargetName,
			}
		}
	}

	if best.slug == "" {
		return fallbackSlug, "padrão de nível"
	}

	desc := fmt.Sprintf("%s (Alvo: %s) │ Vantagem: %.1fx (Risco: %.1fx) │ XP: %d",
		best.name, best.targetName, best.offMult, best.defRisk, best.xp)

	return best.slug, desc
}

// GetNextBestiaryHunt localiza o próximo mapa de caça com abates pendentes para o nível do treinador
// aplicando o Filtro de Sobrevivência Elemental baseado no Pokémon líder
func (c *Catalog) GetNextBestiaryHunt(trainerLevel int, killsBySpecies map[int]int, targetKills int, leaderSpeciesID int) (string, string, int) {
	if targetKills <= 0 {
		targetKills = 120
	}

	type Candidate struct {
		slug            string
		name            string
		minTrainerLevel int
		speciesID       int
		speciesName     string
		kills           int
		survivalTier    int
		isSuicidal      bool
		offMult         float64
		defRisk         float64
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	leader, hasLeader := c.speciesByID[leaderSpeciesID]

	var candidates []Candidate
	for _, h := range c.data.Hunts {
		if h.MinTrainerLevel > trainerLevel || unroutableHunts[h.Slug] {
			continue
		}
		if h.Card == nil || len(h.Card.Species) == 0 {
			continue
		}
		primaryID := h.Card.Species[0].ID
		k := killsBySpecies[primaryID]
		if k < targetKills {
			off := 1.0
			risk := 1.0
			if hasLeader && len(leader.Types) > 0 {
				totalOff := 0.0
				totalRisk := 0.0
				count := 0
				for _, sRef := range h.Card.Species {
					if wild, ok := c.speciesByID[sRef.ID]; ok {
						totalOff += GetBestOffensiveMultiplier(leader.Types, wild.Types)
						totalRisk += GetIncomingRisk(leader.Types, wild.Types)
						count++
					}
				}
				if count > 0 {
					off = totalOff / float64(count)
					risk = totalRisk / float64(count)
				}
			}

			// Classificação de Sobrevivência Elemental
			// isSuicidal: Mobs de nível alto (>=50) onde tomamos dano crítico (>=2.0x) ou tomamos mais de 1.2x e damos menos de 1.0x
			isSuicidal := h.MinTrainerLevel >= 50 && (risk >= 2.0 || (risk > 1.2 && off < 1.0))
			survivalTier := 3 // Padrão neutro

			if off >= 1.5 && risk <= 1.0 {
				survivalTier = 0 // Dominante: Super efetivo ofensivo e defensivamente seguro
			} else if off >= 1.0 && risk <= 1.0 {
				survivalTier = 1 // Seguro neutro
			} else if off >= 1.5 && risk < 2.0 {
				survivalTier = 2 // Alta vantagem ofensiva, risco moderado
			} else if off >= 1.0 && risk < 2.0 {
				survivalTier = 3 // Neutro com risco moderado
			} else if off < 1.0 && risk <= 1.0 {
				survivalTier = 4 // Dano reduzido mas sobrevive com segurança
			} else {
				survivalTier = 5 // Perigoso ou suicida
			}

			candidates = append(candidates, Candidate{
				slug:            h.Slug,
				name:            h.Name,
				minTrainerLevel: h.MinTrainerLevel,
				speciesID:       primaryID,
				speciesName:     c.GetSpeciesName(primaryID),
				kills:           k,
				survivalTier:    survivalTier,
				isSuicidal:      isSuicidal,
				offMult:         off,
				defRisk:         risk,
			})
		}
	}

	if len(candidates) == 0 {
		return "", "", 0
	}

	// Ordenação tática com Filtro de Sobrevivência:
	// 1. Não suicida antes de suicida (evita que o bot fique morrendo em loop contra mobs 4.0x mais fortes)
	// 2. survivalTier ASC (favorece rotas com vantagem super-efetiva de 2.0x a 4.0x)
	// 3. minTrainerLevel ASC (limpa dos níveis mais baixos aos mais altos)
	// 4. Kills DESC (se já tem 80/120 abates, finaliza antes de começar do zero)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].isSuicidal != candidates[j].isSuicidal {
			return !candidates[i].isSuicidal // false (seguro) vem antes de true (suicida)
		}
		if candidates[i].survivalTier != candidates[j].survivalTier {
			return candidates[i].survivalTier < candidates[j].survivalTier
		}
		if candidates[i].minTrainerLevel != candidates[j].minTrainerLevel {
			return candidates[i].minTrainerLevel < candidates[j].minTrainerLevel
		}
		return candidates[i].kills > candidates[j].kills
	})

	picked := candidates[0]
	desc := fmt.Sprintf("%s (%s) [Bestiário: %d/%d abates │ Vantagem: %.1fx (Risco: %.1fx)]",
		picked.name, picked.speciesName, picked.kills, targetKills, picked.offMult, picked.defRisk)
	return picked.slug, desc, len(candidates)
}

