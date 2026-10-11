package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type SocketHandler func(data []byte)

type SocketClient struct {
	wsURL     string
	ticket    string
	conn      *websocket.Conn
	writeMu   sync.Mutex
	closeChan chan struct{}
	closed    bool
	mu        sync.RWMutex

	seq int64

	handlersMu sync.RWMutex
	handlers   map[string][]SocketHandler

	connected bool
	latency   time.Duration
}

func NewSocketClient() *SocketClient {
	return &SocketClient{
		handlers:  make(map[string][]SocketHandler),
		closeChan: make(chan struct{}),
	}
}

func (s *SocketClient) On(eventType string, handler SocketHandler) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	s.handlers[eventType] = append(s.handlers[eventType], handler)
}

func (s *SocketClient) dispatch(eventType string, data []byte) {
	s.handlersMu.RLock()
	handlers := s.handlers[eventType]
	allHandlers := s.handlers["*"]
	s.handlersMu.RUnlock()

	for _, h := range handlers {
		h(data)
	}
	for _, h := range allHandlers {
		h(data)
	}
}

// Connect conecta ao WebSocket do Sigma Project
func (s *SocketClient) Connect(ticket, serverWsURL string) error {
	s.ticket = ticket
	if serverWsURL == "" {
		serverWsURL = "wss://api.poke.sigmaproject.online"
	}

	endpoint := fmt.Sprintf("%s/ws?ticket=%s", serverWsURL, url.QueryEscape(ticket))
	s.wsURL = endpoint

	headers := http.Header{}
	headers.Set("User-Agent", UserAgent)
	headers.Set("Origin", OriginURL)

	dialer := websocket.Dialer{
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: false},
		HandshakeTimeout: 15 * time.Second,
	}

	conn, _, err := dialer.Dial(endpoint, headers)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	s.mu.Lock()
	s.conn = conn
	s.connected = true
	s.closed = false
	s.closeChan = make(chan struct{})
	s.seq = 0
	s.mu.Unlock()

	go s.readPump()
	go s.pingPump()

	return nil
}

// SendIntent envia uma intenção formatada para o servidor {seq, intent}
func (s *SocketClient) SendIntent(intent any) error {
	seq := atomic.AddInt64(&s.seq, 1)
	payload := map[string]any{
		"seq":    seq,
		"intent": intent,
	}
	return s.SendJSON(payload)
}

// SendChat envia uma mensagem de chat
func (s *SocketClient) SendChat(channel, text, to string) error {
	payload := map[string]any{
		"type":    "chat",
		"channel": channel,
		"text":    text,
	}
	if to != "" {
		payload["to"] = to
	}
	return s.SendJSON(payload)
}

// SendJSON envia qualquer payload serializado em JSON
func (s *SocketClient) SendJSON(v any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.mu.RLock()
	conn := s.conn
	closed := s.closed
	s.mu.RUnlock()

	if closed || conn == nil {
		return fmt.Errorf("socket não conectado")
	}

	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(v)
}

func (s *SocketClient) readPump() {
	defer func() {
		s.Close()
	}()

	conn := s.conn
	conn.SetReadLimit(10 * 1024 * 1024) // 10MB para snapshots grandes
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Socket] Conexão fechada com erro: %v", err)
			}
			break
		}

		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// Determina o tipo de mensagem
		var header struct {
			Type     string          `json:"type"`
			Snapshot json.RawMessage `json:"snapshot"`
		}
		if err := json.Unmarshal(message, &header); err == nil {
			if header.Type != "" {
				s.dispatch(header.Type, message)
			}
			if len(header.Snapshot) > 0 {
				s.dispatch("snapshot", message)
			}
		} else {
			s.dispatch("raw", message)
		}
	}
}

func (s *SocketClient) pingPump() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.closeChan:
			return
		case <-ticker.C:
			s.writeMu.Lock()
			s.mu.RLock()
			conn := s.conn
			closed := s.closed
			s.mu.RUnlock()

			if closed || conn == nil {
				s.writeMu.Unlock()
				return
			}

			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.writeMu.Unlock()
				s.Close()
				return
			}
			s.writeMu.Unlock()
		}
	}
}

// IsConnected retorna o estado da conexão
func (s *SocketClient) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected && !s.closed
}

// Close fecha a conexão do socket
func (s *SocketClient) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.connected = false
	close(s.closeChan)
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.mu.Unlock()

	s.dispatch("close", []byte(`{"event":"close"}`))
}
