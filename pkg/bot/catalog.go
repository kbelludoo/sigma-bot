package bot

import (
	_ "embed"
	"encoding/json"
	"fmt"
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

// GetOptimalHunt retorna a rota com maior taxa de XP/Gold permitida para o nível do treinador
func (c *Catalog) GetOptimalHunt(level int) string {
	switch {
	case level >= 100:
		return "electabuzz"
	case level >= 80:
		return "vaporeon"
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
		if h.MinTrainerLevel == maxTier {
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
