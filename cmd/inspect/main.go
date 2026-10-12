package main

import (
	"encoding/json"
	"fmt"
	"time"

	"bot-sigma/pkg/client"
)

func main() {
	c := client.NewAPIClient("")
	login, err := c.Login("kbelludoo@gmail.com", "ECTbb,,0209")
	if err != nil {
		fmt.Printf("Login err: %v\n", err)
		return
	}
	ticket, _ := c.GetTicket(login.AccessToken)

	sock := client.NewSocketClient("")
	sock.On("snapshot", func(data []byte) {
		var msg struct {
			Snapshot struct {
				Trainer struct {
					Gold int `json:"gold"`
				} `json:"trainer"`
				InCity bool `json:"inCity"`
				Hunt   any  `json:"hunt"`
			} `json:"snapshot"`
		}
		_ = json.Unmarshal(data, &msg)
		if msg.Snapshot.Trainer.Gold > 0 || msg.Snapshot.InCity {
			fmt.Printf("Gold: %d, InCity: %v, Hunt: %+v\n", msg.Snapshot.Trainer.Gold, msg.Snapshot.InCity, msg.Snapshot.Hunt)
		}
	})

	_ = sock.Connect(ticket.Ticket, ticket.URL)
	time.Sleep(1 * time.Second)

	fmt.Println("Selling 100x item 140...")
	_ = sock.SendIntent(map[string]any{
		"type":   "sellItem",
		"itemId": 140,
		"count":  100,
	})

	time.Sleep(2 * time.Second)
	sock.Close()
}




