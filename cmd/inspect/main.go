package main

import (
	"fmt"
	"io"
	"net/http"

	"bot-sigma/pkg/client"
)

func main() {
	c := client.NewAPIClient("")

	// Testa envio de aplicação para NOMEKOP
	guildID := "9e8dbe41-fa41-48af-bc2e-f66a4910a1ff" // NOMEKOP [NMK] Nível 11

	applyAccount(c, "kbelludoo@gmail.com", "ECTbb,,0209", guildID, "acc1 (kbelludoo)")
	applyAccount(c, "brunofonsecadesouza89@gmail.com", "ECTbb,,0209", guildID, "acc2 (kbelludoo2)")
	applyAccount(c, "lohmoreira91@gmail.com", "ECTbb,,0209", guildID, "acc3 (LrN)")
}

func applyAccount(c *client.APIClient, email, pass, guildID, label string) {
	login, err := c.Login(email, pass)
	if err != nil {
		fmt.Printf("[%s] Erro login: %v\n", label, err)
		return
	}

	req, _ := http.NewRequest("POST", "https://api.poke.sigmaproject.online/guild/applications/"+guildID, nil)
	req.Header.Set("Authorization", "Bearer "+login.AccessToken)
	req.Header.Set("Origin", "https://poke.sigmaproject.online")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		fmt.Printf("[%s] Erro apply: %v\n", label, err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("[%s] Aplicação para NOMEKOP -> Status: %d, Resposta: %s\n", label, resp.StatusCode, string(body))
}
