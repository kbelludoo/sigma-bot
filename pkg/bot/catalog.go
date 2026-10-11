package bot

import (
	_ "embed"
	"encoding/json"
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

type CatalogHunt struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	MinTrainerLevel int    `json:"minTrainerLevel"`
	Region          string `json:"region"`
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
