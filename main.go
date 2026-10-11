package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"bot-sigma/pkg/bot"
	"bot-sigma/pkg/server"
)

//go:embed accounts.json
var defaultAccountsJSON []byte

type BotManager struct {
	bots    []*bot.Bot
	botByID map[string]*bot.Bot
}

func (m *BotManager) GetAllStatuses() []bot.AccountStatus {
	var list []bot.AccountStatus
	for _, b := range m.bots {
		list = append(list, b.GetStatus())
	}
	return list
}

func (m *BotManager) GetBot(id string) *bot.Bot {
	return m.botByID[id]
}

func formatGold(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d gp", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fk gp", float64(n)/1000.0)
	}
	return fmt.Sprintf("%.2fkk gp", float64(n)/1_000_000.0)
}

func main() {
	log.Println("================================================================================")
	log.Println("    ⚡ SIGMA PROJECT MULTI-BOT NATIVO EM GO (HIGH-PERFORMANCE / ULTRA-LEVE) ⚡    ")
	log.Println("================================================================================")

	// 1. Carregar contas do accounts.json
	var accountConfigs []bot.AccountConfig

	content := defaultAccountsJSON
	if diskData, err := os.ReadFile("accounts.json"); err == nil && len(diskData) > 0 {
		content = diskData
	} else {
		_ = os.WriteFile("accounts.json", defaultAccountsJSON, 0644)
	}

	if err := json.Unmarshal(content, &accountConfigs); err != nil {
		log.Fatalf("Falha ao analisar accounts.json: %v", err)
	}

	targetAcc := os.Getenv("ACCOUNT")
	var activeConfigs []bot.AccountConfig

	for _, a := range accountConfigs {
		if !a.Enabled && a.ID != targetAcc {
			continue
		}
		if targetAcc != "" && a.ID != targetAcc {
			continue
		}
		activeConfigs = append(activeConfigs, a)
	}

	if len(activeConfigs) == 0 {
		log.Printf("⚠️ Nenhuma conta com 'enabled: true' configurada no accounts.json.")
		log.Printf("Edite o arquivo 'accounts.json' com seus logins e defina 'enabled: true'.")
		// Continua para que o servidor web suba permitindo visualização
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := &BotManager{
		botByID: make(map[string]*bot.Bot),
	}

	log.Printf("Iniciando %d conta(s) simultânea(s) em Goroutines nativas…\n", len(activeConfigs))

	for _, cfg := range activeConfigs {
		b := bot.NewBot(cfg)
		manager.bots = append(manager.bots, b)
		manager.botByID[cfg.ID] = b
		go b.Run(ctx)
	}

	// 2. Servidor Web Dashboard (Porta 3002 padrão, ou variável de ambiente PORT)
	port := 3002
	if pStr := os.Getenv("PORT"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil {
			port = p
		}
	}

	srv := server.NewServer(port, manager)
	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("[Web Server] Erro: %v", err)
		}
	}()

	// 3. Monitor Periódico no Terminal com estatísticas de RAM
	go func() {
		time.Sleep(2 * time.Second)
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				allocMB := float64(m.Alloc) / 1024.0 / 1024.0
				sysMB := float64(m.Sys) / 1024.0 / 1024.0

				statuses := manager.GetAllStatuses()
				totalGold := 0
				totalKills := 0
				totalCaptures := 0
				totalShinies := 0
				inGameCount := 0

				for _, st := range statuses {
					totalGold += st.Gold
					totalKills += st.Kills
					totalCaptures += st.Captures
					totalShinies += st.Shinies
					if st.InGame {
						inGameCount++
					}
				}

				fmt.Printf("\n\033[1;36m================================================================================\033[0m\n")
				fmt.Printf("\033[1;36m   ⚡ SIGMA GO ENGINE — RAM: %.1f MB (Sys: %.1f MB) — Goroutines: %d — Ativas: %d/%d ⚡\033[0m\n",
					allocMB, sysMB, runtime.NumGoroutine(), inGameCount, len(statuses))
				fmt.Printf("\033[1;36m================================================================================\033[0m\n")

				for _, st := range statuses {
					statusBadge := "\033[1;33m[ CONECTANDO ]\033[0m"
					if st.InGame {
						statusBadge = "\033[1;32m[ EM JOGO ]\033[0m"
					}

					name := st.TrainerName
					if name == "" {
						name = st.Alias
					}
					if name == "" {
						name = st.ID
					}

					fmt.Printf("[%s: \033[1m%s\033[0m (Nv. %d)] %s\n", st.ID, name, st.Level, statusBadge)
					fmt.Printf("  🗺️  Rota: \033[1;37m%s\033[0m │ ❤️  Vida: %d/%d │ ⚔️  Kills: %d │ 🎯 Caps: %d │ ✨ Shinies: %d\n",
						st.MapName, st.HP, st.MaxHP, st.Kills, st.Captures, st.Shinies)
					fmt.Printf("  💰 Ouro: \033[1;33m%s\033[0m (%d gp) │ 💎 Diamantes: %d │ XP: %d/%d (%.1f%%)\n",
						formatGold(st.Gold), st.Gold, st.Diamonds, st.XP, st.XPToNextLevel, st.XPPercent)
					if st.LeaderName != "" {
						fmt.Printf("  🐾 Líder: %s (Nv. %d) │ 🎯 Alvo: %s (Nv. %d)\n",
							st.LeaderName, st.LeaderLevel, st.WildTarget, st.WildLevel)
					}
					if st.LastAction != "" {
						fmt.Printf("  📝 Ação: \033[0;90m%s\033[0m\n", st.LastAction)
					}
				}
			}
		}
	}()

	// 4. Capturar sinais para finalização graciosa
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("\n[Finalizando] Encerrando todas as instâncias do bot graciosamente…")
	cancel()
	time.Sleep(500 * time.Millisecond)
	log.Println("[Finalizado] Até logo!")
}
