package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"bot-sigma/pkg/bot"
	"bot-sigma/pkg/client"
)

func main() {
	cat := bot.GetGlobalCatalog()
	c := client.NewAPIClient("")
	login, err := c.Login("kbelludoo@gmail.com", "ECTbb,,0209")
	if err != nil {
		fmt.Printf("Login err: %v\n", err)
		return
	}

	req, _ := http.NewRequest("GET", "https://api.poke.sigmaproject.online/bestiary", nil)
	req.Header.Set("Authorization", "Bearer "+login.AccessToken)
	req.Header.Set("Origin", "https://poke.sigmaproject.online")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, _ := (&http.Client{}).Do(req)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var besData struct {
		Species []struct {
			ID    int `json:"id"`
			Kills int `json:"kills"`
			Level int `json:"level"`
		} `json:"species"`
	}
	_ = json.Unmarshal(body, &besData)

	killsBySpecies := make(map[int]int)
	for _, s := range besData.Species {
		killsBySpecies[s.ID] = s.Kills
	}

	hunts := cat.GetAllHunts()
	fmt.Printf("=== MAPAS DE CAÇA E ABATES (TOTAL HUNTS: %d) ===\n", len(hunts))
	incomplete := 0
	for _, h := range hunts {
		if h.MinTrainerLevel <= 80 && h.Card != nil && len(h.Card.Species) > 0 {
			primaryID := h.Card.Species[0].ID
			kills := killsBySpecies[primaryID]
			status := "✓ PRONTO"
			if kills < 120 {
				status = fmt.Sprintf("PENDENTE (%d/120)", kills)
				incomplete++
			}
			fmt.Printf("MinLv: %2d | Slug: %-15s | Monstro: %-15s | Kills: %5d | %s\n",
				h.MinTrainerLevel, h.Slug, cat.GetSpeciesName(primaryID), kills, status)
		}
	}
	fmt.Printf("\nTotal de mapas pendentes (< 120 abates): %d\n", incomplete)
}
