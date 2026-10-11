package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"bot-sigma/pkg/client"
)

var IPMap = map[string]string{
	"10.0.0.191": "SEU_IP_DA_VPS",
	"10.0.0.49":  "141.148.34.23",
	"10.0.0.217": "193.122.133.1",
	"10.0.0.79":  "129.80.45.1",
}

type Bot struct {
	config AccountConfig

	mu     sync.RWMutex
	status AccountStatus

	apiClient *client.APIClient
	socket    *client.SocketClient
	catalog   *Catalog

	targetMapSlug  string
	paused         bool
	policyApplied  bool
	lastHealTime   time.Time
	lastClaimTime   time.Time
	lastTravelTime  time.Time
	lastSpinTime    time.Time
	lastTargetTime  time.Time
	lastBallBuyTime time.Time
	totalBalls      int

	initialGold int
	initialXP   int
	hasInitial  bool
}

func NewBot(cfg AccountConfig) *Bot {
	cat := GetGlobalCatalog()

	ballID := cfg.BallID
	if ballID <= 0 {
		ballID = 1 // Poke Ball
	}

	shinyBallID := cfg.ShinyBallID
	if shinyBallID <= 0 {
		shinyBallID = 5 // Master Ball
	}

	cfg.BallID = ballID
	cfg.ShinyBallID = shinyBallID

	outbound := "Padrão"
	if pub, ok := IPMap[cfg.BindIP]; ok {
		outbound = pub
	} else if cfg.BindIP != "" {
		outbound = cfg.BindIP
	}

	return &Bot{
		config:        cfg,
		apiClient:     client.NewAPIClient(cfg.BindIP),
		catalog:       cat,
		targetMapSlug: cfg.HuntSlug,
		status: AccountStatus{
			ID:          cfg.ID,
			Alias:       cfg.Name,
			TrainerName: cfg.Name,
			BindIP:      cfg.BindIP,
			OutboundIP:  outbound,
			MapSlug:     cfg.HuntSlug,
			MapName:     cat.GetHuntName(cfg.HuntSlug),
			LastAction:  "Inicializado",
			UpdatedAt:   time.Now().UnixMilli(),
		},
	}
}

func (b *Bot) GetConfig() AccountConfig {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.config
}

func (b *Bot) GetStatus() AccountStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	st := b.status
	if st.MaxHP > 0 {
		st.HPPercent = float64(st.HP) / float64(st.MaxHP) * 100.0
	}
	if st.XPToNextLevel > 0 {
		st.XPPercent = float64(st.XP) / float64(st.XPToNextLevel) * 100.0
	}
	return st
}

func (b *Bot) persistStatus() {
	st := b.GetStatus()
	st.UpdatedAt = time.Now().UnixMilli()

	_ = os.MkdirAll("data", 0755)
	filename := filepath.Join("data", fmt.Sprintf("status_%s.json", b.config.ID))
	data, err := json.MarshalIndent(st, "", "  ")
	if err == nil {
		_ = os.WriteFile(filename, data, 0644)
	}
}

// Run inicia o ciclo de vida da conta com reconexão resiliente
func (b *Bot) Run(ctx context.Context) {
	// Persistência periódica do status
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.persistStatus()
			}
		}
	}()

	backoff := 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			b.disconnect()
			return
		default:
		}

		startTime := time.Now()
		err := b.runSession(ctx)
		if err != nil {
			b.mu.Lock()
			b.status.InGame = false
			b.status.LastError = err.Error()
			b.status.LastAction = fmt.Sprintf("Erro: %v (tentando em %v)", err, backoff)
			b.mu.Unlock()

			log.Printf("[%s] Sessão encerrou: %v. Próxima tentativa em %v…", b.config.ID, err, backoff)
		}

		// Se a sessão durou mais de 1 minuto, reseta o backoff
		if time.Since(startTime) > 1*time.Minute {
			backoff = 5 * time.Second
		} else {
			backoff += 5 * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
	}
}

func (b *Bot) disconnect() {
	b.mu.Lock()
	if b.socket != nil {
		b.socket.Close()
	}
	b.status.InGame = false
	b.mu.Unlock()
}

func (b *Bot) runSession(ctx context.Context) error {
	b.mu.Lock()
	b.policyApplied = false
	b.status.InGame = false
	b.status.LastError = ""
	b.status.LastAction = "Autenticando..."
	b.mu.Unlock()

	log.Printf("[%s] Fazendo login para %s (%s)…", b.config.ID, b.config.Name, b.config.Email)
	loginResp, err := b.apiClient.Login(b.config.Email, b.config.Password)
	if err != nil {
		return fmt.Errorf("falha no login: %w", err)
	}

	b.mu.Lock()
	b.status.LastAction = "Obtendo ticket do servidor..."
	b.mu.Unlock()

	ticketResp, err := b.apiClient.GetTicket(loginResp.AccessToken)
	if err != nil {
		return fmt.Errorf("falha ao obter ticket: %w", err)
	}

	sock := client.NewSocketClient(b.config.BindIP)
	b.socket = sock

	disconnectChan := make(chan struct{}, 1)

	// Registra ouvintes do WebSocket
	b.setupHandlers(sock, disconnectChan)

	b.mu.Lock()
	b.status.LastAction = "Conectando WebSocket..."
	b.mu.Unlock()

	err = sock.Connect(ticketResp.Ticket, ticketResp.URL)
	if err != nil {
		return fmt.Errorf("falha ao conectar WebSocket: %w", err)
	}

	b.mu.Lock()
	b.status.InGame = true
	b.status.ConnectedAt = time.Now().UnixMilli()
	b.status.LastAction = "Conectado. Aguardando mundo..."
	b.mu.Unlock()

	log.Printf("[%s: %s] Conectado ao jogo com sucesso! ✓", b.config.ID, b.config.Name)

	// Executa a rotina de automação periódica em goroutine
	go b.automationLoop(ctx, sock, disconnectChan)

	// Aguarda queda da conexão ou cancelamento do contexto
	select {
	case <-ctx.Done():
		b.disconnect()
		return nil
	case <-disconnectChan:
		return fmt.Errorf("conexão websocket encerrada")
	}
}

func (b *Bot) setupHandlers(sock *client.SocketClient, disconnectChan chan struct{}) {
	sock.On("snapshot", func(data []byte) {
		var msg SnapshotMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		if msg.Snapshot != nil {
			b.updateSnapshot(msg.Snapshot)
		}
		if len(msg.Events) > 0 {
			b.processEvents(msg.Events)
		}
	})

	sock.On("notice", func(data []byte) {
		var notice struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(data, &notice); err == nil && notice.Message != "" {
			b.mu.Lock()
			b.status.LastAction = fmt.Sprintf("Aviso: %s", notice.Message)
			b.mu.Unlock()
		}
	})

	sock.On("rareCapture", func(data []byte) {
		var capture struct {
			TrainerName string `json:"trainerName"`
			SpeciesID   int    `json:"speciesId"`
			Shiny       bool   `json:"shiny"`
		}
		if err := json.Unmarshal(data, &capture); err == nil {
			monName := b.catalog.GetSpeciesName(capture.SpeciesID)
			log.Printf("⭐ [CAPTURA RARA] %s capturou %s (Shiny: %v)!", capture.TrainerName, monName, capture.Shiny)
		}
	})

	sock.On("fatal", func(data []byte) {
		var fatal struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &fatal)
		log.Printf("[%s] Erro fatal do servidor: %s (%s)", b.config.ID, fatal.Message, fatal.Code)
		b.mu.Lock()
		b.status.LastError = fatal.Message
		b.mu.Unlock()
		select {
		case disconnectChan <- struct{}{}:
		default:
		}
	})

	sock.On("close", func([]byte) {
		select {
		case disconnectChan <- struct{}{}:
		default:
		}
	})
}

func (b *Bot) updateSnapshot(snap *GameStateData) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.status.TrainerName = snap.Trainer.Name
	b.status.Level = snap.Trainer.Level
	b.status.XP = snap.Trainer.XP
	b.status.XPToNextLevel = snap.Trainer.XPToNextLevel
	b.status.HP = snap.Trainer.HP
	b.status.MaxHP = snap.Trainer.MaxHP
	b.status.Gold = snap.Trainer.Gold
	b.status.Diamonds = snap.Trainer.Diamonds
	b.status.AutoHuntActive = snap.AutoHunt

	if !b.hasInitial {
		b.initialGold = snap.Trainer.Gold
		b.initialXP = snap.Trainer.XP
		b.hasInitial = true
	} else {
		if snap.Trainer.Gold >= b.initialGold {
			b.status.GoldFound = snap.Trainer.Gold - b.initialGold
		}
		if snap.Trainer.XP >= b.initialXP {
			b.status.TrainerXPGained = snap.Trainer.XP - b.initialXP
		}
	}

	// Estatísticas do Analisador interno
	if snap.Analyzer != nil {
		b.status.Kills = snap.Analyzer.Kills
		b.status.Captures = snap.Analyzer.Captures
		b.status.Shinies = snap.Analyzer.ShinyCaptures
		if snap.Analyzer.GoldFound > 0 {
			b.status.GoldFound = snap.Analyzer.GoldFound
		}
		if snap.Analyzer.TrainerXP > 0 {
			b.status.TrainerXPGained = snap.Analyzer.TrainerXP
		}
		if snap.Analyzer.ElapsedMs >= 3000 {
			mins := float64(snap.Analyzer.ElapsedMs) / 60000.0
			b.status.KillsPerMin = float64(snap.Analyzer.Kills) / mins
			b.status.XPPerHour = (float64(snap.Analyzer.TrainerXP) / mins) * 60.0
		} else {
			elapsedMins := float64(time.Now().UnixMilli()-b.status.ConnectedAt) / 60000.0
			if elapsedMins > 0.1 {
				b.status.KillsPerMin = float64(b.status.Kills) / elapsedMins
				b.status.XPPerHour = (float64(b.status.TrainerXPGained) / elapsedMins) * 60.0
			}
		}
	}

	// Pokémon Líder da equipe
	if snap.Leader != nil {
		b.status.LeaderSpeciesID = snap.Leader.SpeciesID
		b.status.LeaderName = b.catalog.GetSpeciesName(snap.Leader.SpeciesID)
		b.status.LeaderLevel = snap.Leader.Level
		b.status.LeaderHP = snap.Leader.HP
		b.status.LeaderMaxHP = snap.Leader.MaxHP
		b.status.LeaderShiny = snap.Leader.Shiny
	} else if len(snap.Team) > 0 {
		leader := snap.Team[0]
		b.status.LeaderSpeciesID = leader.SpeciesID
		b.status.LeaderName = b.catalog.GetSpeciesName(leader.SpeciesID)
		b.status.LeaderLevel = leader.Level
		b.status.LeaderHP = leader.HP
		b.status.LeaderMaxHP = leader.MaxHP
		b.status.LeaderShiny = leader.Shiny
	}

	// Dados de caça e mapa
	if snap.Hunt != nil {
		b.status.MapSlug = snap.Hunt.MapSlug
		b.status.MapName = b.catalog.GetHuntName(snap.Hunt.MapSlug)

		if snap.Hunt.WildSpeciesID > 0 {
			b.status.WildTarget = b.catalog.GetSpeciesName(snap.Hunt.WildSpeciesID)
			b.status.WildHP = snap.Hunt.WildHP
			b.status.WildMaxHP = snap.Hunt.WildMaxHP
			b.status.WildLevel = snap.Hunt.WildLevel
			b.status.WildShiny = snap.Hunt.WildShiny
		} else if len(snap.Hunt.Wilds) > 0 {
			w := snap.Hunt.Wilds[0]
			b.status.WildTarget = b.catalog.GetSpeciesName(w.SpeciesID)
			b.status.WildHP = w.HP
			b.status.WildMaxHP = w.MaxHP
			b.status.WildLevel = w.Level
			b.status.WildShiny = w.Shiny
		}
	}

	// Contagem total de Pokébolas
	totalB := 0
	for _, cnt := range snap.Balls {
		totalB += cnt
	}
	for _, cnt := range snap.BoundBalls {
		totalB += cnt
	}
	b.totalBalls = totalB

	b.status.UpdatedAt = time.Now().UnixMilli()
}

func (b *Bot) processEvents(events []EventData) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ev := range events {
		switch ev.Type {
		case "kill":
			b.status.Kills++
			mon := b.catalog.GetSpeciesName(ev.SpeciesID)
			b.status.LastAction = fmt.Sprintf("Derrotou %s (+%d XP, +%d Ouro)", mon, ev.XP, ev.Gold)
		case "captured":
			b.status.Captures++
			mon := b.catalog.GetSpeciesName(ev.SpeciesID)
			if ev.Shiny {
				b.status.Shinies++
				log.Printf("✨ [%s] POKÉMON SHINY CAPTURADO: %s! ✨", b.config.ID, mon)
				b.status.LastAction = fmt.Sprintf("✨ Capturou SHINY %s!", mon)
			} else {
				b.status.LastAction = fmt.Sprintf("Capturou %s com sucesso!", mon)
			}
		case "shinyAppeared":
			mon := b.catalog.GetSpeciesName(ev.SpeciesID)
			log.Printf("⭐ [%s] SHINY AVISTADO NA ROTA: %s! ⭐", b.config.ID, mon)
			b.status.LastAction = fmt.Sprintf("⭐ Shiny avistado: %s!", mon)
		case "healed":
			b.status.LastAction = "Curado no Centro Pokémon"
		case "levelUp":
			b.status.Level = ev.Level
			b.status.LastAction = fmt.Sprintf("Subiu para o Nível %d!", ev.Level)
			log.Printf("🎉 [%s] Treinador subiu para o Nível %d! 🎉", b.config.ID, ev.Level)
		}
	}
}

// automationLoop executa os comandos periódicos inteligentes
func (b *Bot) automationLoop(ctx context.Context, sock *client.SocketClient, disconnectChan chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-disconnectChan:
			return
		case <-ticker.C:
			if !sock.IsConnected() {
				return
			}
			b.tickAutomation(sock)
		}
	}
}

func (b *Bot) tickAutomation(sock *client.SocketClient) {
	b.mu.Lock()
	if b.paused {
		b.status.LastAction = "Pausado pelo usuário"
		b.mu.Unlock()
		return
	}

	targetMap := b.targetMapSlug
	currentMap := b.status.MapSlug
	hp := b.status.HP
	maxHP := b.status.MaxHP
	leaderHP := b.status.LeaderHP
	cfg := b.config
	b.mu.Unlock()

	// Política nativa do servidor já cuida de auto-catch e auto-hunt perfeitamente.

	// 2. Viagem inteligente: se huntSlug não for fixo ou for "auto", calcula a melhor rota considerando o tipo elemental do Líder
	destMap := targetMap
	tacticalReason := ""
	if destMap == "" || destMap == "auto" {
		destMap, tacticalReason = b.catalog.GetOptimalHuntForLeader(b.status.Level, b.status.LeaderSpeciesID)
	}

	if destMap != "" && currentMap != "" && destMap != currentMap && time.Since(b.lastTravelTime) > 12*time.Second {
		b.lastTravelTime = time.Now()
		log.Printf("[%s] 🚀 [ROTA TÁTICA ELEMENTAL] Viajando de '%s' para: '%s' (%s)…", b.config.ID, currentMap, destMap, tacticalReason)
		_ = sock.SendIntent(TravelIntent{
			Type:    "travel",
			MapSlug: destMap,
		})
		b.mu.Lock()
		b.status.LastAction = fmt.Sprintf("Viajando para %s (%s)...", b.catalog.GetHuntName(destMap), tacticalReason)
		b.mu.Unlock()
		return
	}

	// 3. Auto-Heal (Curar no Centro Pokémon se HP estiver baixo ou desmaiado)
	if cfg.AutoHeal && time.Since(b.lastHealTime) > 10*time.Second {
		lowHealth := (maxHP > 0 && float64(hp)/float64(maxHP) < 0.25) || (leaderHP == 0 && maxHP > 0)
		if lowHealth {
			b.lastHealTime = time.Now()
			log.Printf("[%s] Vida baixa detectada (%d/%d). Enviando healAtCenter…", b.config.ID, hp, maxHP)
			_ = sock.SendIntent(HealAtCenterIntent{
				Type: "healAtCenter",
			})
			b.mu.Lock()
			b.status.LastAction = "Curando no Centro Pokémon..."
			b.mu.Unlock()
			return
		}
	}

	// 4. Garantir que o AutoHunt nativo do jogo esteja ativo
	if cfg.AutoHunt && !b.status.AutoHuntActive {
		_ = sock.SendIntent(SetAutoHuntIntent{
			Type: "setAutoHunt",
			On:   true,
		})
		b.mu.Lock()
		b.status.AutoHuntActive = true
		b.status.LastAction = "Ativando Auto-Hunt..."
		b.mu.Unlock()
	}

	// 5. Coleta automática de login diário, recompensas de quests, bestiário e tarefas (a cada 45 segundos)
	if cfg.AutoDaily && time.Since(b.lastClaimTime) > 45*time.Second {
		b.lastClaimTime = time.Now()
		_ = sock.SendIntent(SimpleIntent{Type: "claimDailyLogin"})
		_ = sock.SendIntent(SimpleIntent{Type: "claimQuestReward"})
		_ = sock.SendIntent(SimpleIntent{Type: "claimBestiaryAll"})
		_ = sock.SendIntent(SimpleIntent{Type: "claimDailyTasks"})
		_ = sock.SendIntent(SimpleIntent{Type: "claimWeeklyTasks"})
		b.mu.Lock()
		b.status.LastAction = "Coletando recompensas diárias..."
		b.mu.Unlock()
	}

	// 6. Girar roda da sorte se disponível (a cada 60 segundos)
	if cfg.AutoSpin && time.Since(b.lastSpinTime) > 60*time.Second {
		b.lastSpinTime = time.Now()
		_ = sock.SendIntent(SpinWheelIntent{
			Type:    "spinWheel",
			WheelID: "daily",
			Times:   1,
		})
	}

	// 7. Auto-compra de Pokébolas se estiver com poucas bolas (< 10) e tiver ouro
	if b.totalBalls < 10 && b.status.Gold >= 200 && time.Since(b.lastBallBuyTime) > 30*time.Second {
		b.lastBallBuyTime = time.Now()
		log.Printf("[%s] 🛒 [AUTO-COMPRA] Pokébolas acabando (%d restantes). Comprando 20 Pokébolas no NPC…", b.config.ID, b.totalBalls)
		_ = sock.SendIntent(map[string]any{
			"type":   "buyBall",
			"ballId": 1,
			"count":  20,
		})
	}
}

// Métodos de controle externo pelo Web Dashboard

func (b *Bot) TravelTo(mapSlug string) error {
	b.mu.Lock()
	b.targetMapSlug = mapSlug
	b.config.HuntSlug = mapSlug
	sock := b.socket
	b.mu.Unlock()

	if sock != nil && sock.IsConnected() {
		return sock.SendIntent(TravelIntent{
			Type:    "travel",
			MapSlug: mapSlug,
		})
	}
	return nil
}

func (b *Bot) ToggleAutoHunt(on bool) error {
	b.mu.Lock()
	b.config.AutoHunt = on
	sock := b.socket
	b.mu.Unlock()

	if sock != nil && sock.IsConnected() {
		return sock.SendIntent(SetAutoHuntIntent{
			Type: "setAutoHunt",
			On:   on,
		})
	}
	return nil
}

func (b *Bot) HealNow() error {
	b.mu.Lock()
	sock := b.socket
	b.mu.Unlock()

	if sock != nil && sock.IsConnected() {
		return sock.SendIntent(HealAtCenterIntent{
			Type: "healAtCenter",
		})
	}
	return nil
}

func (b *Bot) ClaimRewardsNow() error {
	b.mu.Lock()
	sock := b.socket
	b.mu.Unlock()

	if sock != nil && sock.IsConnected() {
		_ = sock.SendIntent(SimpleIntent{Type: "claimDailyLogin"})
		_ = sock.SendIntent(SimpleIntent{Type: "claimQuestReward"})
		_ = sock.SendIntent(SimpleIntent{Type: "claimBestiaryAll"})
		_ = sock.SendIntent(SimpleIntent{Type: "claimDailyTasks"})
		_ = sock.SendIntent(SimpleIntent{Type: "claimWeeklyTasks"})
		return nil
	}
	return nil
}

func (b *Bot) Pause() {
	b.mu.Lock()
	b.paused = true
	b.mu.Unlock()
}

func (b *Bot) Resume() {
	b.mu.Lock()
	b.paused = false
	b.mu.Unlock()
}
