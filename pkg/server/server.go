package server

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"bot-sigma/pkg/bot"
)

//go:embed index.html
var defaultIndexHTML []byte

type BotProvider interface {
	GetAllStatuses() []bot.AccountStatus
	GetBot(id string) *bot.Bot
}

type Server struct {
	port     int
	provider BotProvider
	catalog  *bot.Catalog
}

func NewServer(port int, provider BotProvider) *Server {
	return &Server{
		port:     port,
		provider: provider,
		catalog:  bot.GetGlobalCatalog(),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// 1. API Status
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		allocMB := float64(m.Alloc) / 1024.0 / 1024.0

		statuses := s.provider.GetAllStatuses()

		resp := map[string]any{
			"serverTime": time.Now().UnixMilli(),
			"ramMB":      allocMB,
			"accounts":   statuses,
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	// 2. API Hunts (Rotas e monstros disponíveis)
	mux.HandleFunc("/api/hunts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		hunts := s.catalog.GetAllHunts()
		_ = json.NewEncoder(w).Encode(hunts)
	})

	// 3. API Action (Controles remotos)
	mux.HandleFunc("/api/action", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			BotID       string `json:"botId"`
			Action      string `json:"action"`
			MapSlug     string `json:"mapSlug,omitempty"`
			On          bool   `json:"on,omitempty"`
			Strategy    string `json:"strategy,omitempty"`
			TargetKills int    `json:"targetKills,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}

		b := s.provider.GetBot(req.BotID)
		if b == nil {
			http.Error(w, "Bot não encontrado", http.StatusNotFound)
			return
		}

		switch req.Action {
		case "travel":
			if req.MapSlug != "" {
				_ = b.TravelTo(req.MapSlug)
			}
		case "heal":
			_ = b.HealNow()
		case "claim":
			_ = b.ClaimRewardsNow()
		case "toggle-autohunt":
			_ = b.ToggleAutoHunt(req.On)
		case "set-strategy":
			if req.Strategy != "" {
				b.SetStrategy(req.Strategy, req.TargetKills)
			}
		case "pause":
			b.Pause()
		case "resume":
			b.Resume()
		default:
			http.Error(w, "Ação desconhecida", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	// 4. Servir Dashboard Web estático
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(defaultIndexHTML)
	})

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[Web Dashboard] Servidor iniciado em http://localhost%s", addr)
	log.Printf("[Web Dashboard] API pública: http://localhost%s/api/status", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return srv.ListenAndServe()
}
