package bot

import (
	"strings"
)

// Matriz oficial de efetividade de tipos Pokémon (Gerações 6+)
var typeChart = map[string]map[string]float64{
	"NORMAL": {
		"ROCK": 0.5, "STEEL": 0.5, "GHOST": 0.0,
	},
	"FIRE": {
		"FIRE": 0.5, "WATER": 0.5, "GRASS": 2.0, "ICE": 2.0,
		"BUG": 2.0, "ROCK": 0.5, "DRAGON": 0.5, "STEEL": 2.0,
	},
	"WATER": {
		"FIRE": 2.0, "WATER": 0.5, "GRASS": 0.5, "GROUND": 2.0,
		"ROCK": 2.0, "DRAGON": 0.5,
	},
	"GRASS": {
		"FIRE": 0.5, "WATER": 2.0, "GRASS": 0.5, "POISON": 0.5,
		"GROUND": 2.0, "FLYING": 0.5, "BUG": 0.5, "ROCK": 2.0,
		"DRAGON": 0.5, "STEEL": 0.5,
	},
	"ELECTRIC": {
		"WATER": 2.0, "ELECTRIC": 0.5, "GRASS": 0.5, "GROUND": 0.0,
		"FLYING": 2.0, "DRAGON": 0.5,
	},
	"ICE": {
		"FIRE": 0.5, "WATER": 0.5, "GRASS": 2.0, "ICE": 0.5,
		"GROUND": 2.0, "FLYING": 2.0, "DRAGON": 2.0, "STEEL": 0.5,
	},
	"FIGHTING": {
		"NORMAL": 2.0, "ICE": 2.0, "POISON": 0.5, "FLYING": 0.5,
		"PSYCHIC": 0.5, "BUG": 0.5, "ROCK": 2.0, "GHOST": 0.0,
		"DARK": 2.0, "STEEL": 2.0, "FAIRY": 0.5,
	},
	"POISON": {
		"GRASS": 2.0, "POISON": 0.5, "GROUND": 0.5, "ROCK": 0.5,
		"GHOST": 0.5, "STEEL": 0.0, "FAIRY": 2.0,
	},
	"GROUND": {
		"FIRE": 2.0, "ELECTRIC": 2.0, "GRASS": 0.5, "POISON": 2.0,
		"FLYING": 0.0, "BUG": 0.5, "ROCK": 2.0, "STEEL": 2.0,
	},
	"FLYING": {
		"ELECTRIC": 0.5, "GRASS": 2.0, "FIGHTING": 2.0, "BUG": 2.0,
		"ROCK": 0.5, "STEEL": 0.5,
	},
	"PSYCHIC": {
		"FIGHTING": 2.0, "POISON": 2.0, "PSYCHIC": 0.5, "DARK": 0.0,
		"STEEL": 0.5,
	},
	"BUG": {
		"FIRE": 0.5, "GRASS": 2.0, "FIGHTING": 0.5, "POISON": 0.5,
		"FLYING": 0.5, "PSYCHIC": 2.0, "GHOST": 0.5, "STEEL": 0.5,
		"FAIRY": 0.5, "DARK": 2.0,
	},
	"ROCK": {
		"FIRE": 2.0, "ICE": 2.0, "FIGHTING": 0.5, "GROUND": 0.5,
		"FLYING": 2.0, "BUG": 2.0, "STEEL": 0.5,
	},
	"GHOST": {
		"NORMAL": 0.0, "PSYCHIC": 2.0, "GHOST": 2.0, "DARK": 0.5,
	},
	"DRAGON": {
		"DRAGON": 2.0, "STEEL": 0.5, "FAIRY": 0.0,
	},
	"STEEL": {
		"FIRE": 0.5, "WATER": 0.5, "ELECTRIC": 0.5, "ICE": 2.0,
		"ROCK": 2.0, "STEEL": 0.5, "FAIRY": 2.0,
	},
	"FAIRY": {
		"FIRE": 0.5, "FIGHTING": 2.0, "POISON": 0.5, "DRAGON": 2.0,
		"DARK": 2.0, "STEEL": 0.5,
	},
	"DARK": {
		"FIGHTING": 0.5, "PSYCHIC": 2.0, "GHOST": 2.0, "DARK": 0.5,
		"FAIRY": 0.5,
	},
}

// GetSingleTypeEffectiveness retorna a efetividade de um único tipo de ataque contra um tipo defensor
func GetSingleTypeEffectiveness(attackType, defenseType string) float64 {
	atk := strings.ToUpper(attackType)
	def := strings.ToUpper(defenseType)

	if targets, ok := typeChart[atk]; ok {
		if mult, exists := targets[def]; exists {
			return mult
		}
	}
	return 1.0 // Neutro
}

// GetAttackMultiplier calcula a efetividade de um golpe do tipo attackType contra um Pokémon com múltiplos tipos
func GetAttackMultiplier(attackType string, defenderTypes []string) float64 {
	total := 1.0
	for _, dt := range defenderTypes {
		total *= GetSingleTypeEffectiveness(attackType, dt)
	}
	return total
}

// GetBestOffensiveMultiplier calcula o melhor dano que o atacante pode causar no defensor
func GetBestOffensiveMultiplier(attackerTypes, defenderTypes []string) float64 {
	if len(attackerTypes) == 0 {
		return 1.0
	}
	best := 0.0
	for _, at := range attackerTypes {
		mult := GetAttackMultiplier(at, defenderTypes)
		if mult > best {
			best = mult
		}
	}
	return best
}

// GetIncomingRisk calcula o maior dano que o atacante pode receber do defensor
func GetIncomingRisk(attackerTypes, defenderTypes []string) float64 {
	if len(defenderTypes) == 0 {
		return 1.0
	}
	maxRisk := 0.0
	for _, dt := range defenderTypes {
		risk := GetAttackMultiplier(dt, attackerTypes)
		if risk > maxRisk {
			maxRisk = risk
		}
	}
	if maxRisk <= 0.0 {
		maxRisk = 0.25
	}
	return maxRisk
}
