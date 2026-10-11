package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"
)

const (
	BaseURL   = "https://api.poke.sigmaproject.online"
	OriginURL = "https://poke.sigmaproject.online"
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

type APIClient struct {
	httpClient *http.Client
}

func NewAPIClient(bindIP string) *APIClient {
	jar, _ := cookiejar.New(nil)
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if bindIP != "" {
		if ip := net.ParseIP(bindIP); ip != nil {
			transport.DialContext = (&net.Dialer{
				LocalAddr: &net.TCPAddr{IP: ip},
				Timeout:   15 * time.Second,
			}).DialContext
		}
	}

	return &APIClient{
		httpClient: &http.Client{
			Jar:       jar,
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresIn"`
	HasTrainer  bool   `json:"hasTrainer"`
	Error       string `json:"error,omitempty"`
	Code        string `json:"code,omitempty"`
}

type TicketResponse struct {
	Ticket string `json:"ticket"`
	URL    string `json:"url,omitempty"`
}

type TrainerInfo struct {
	Trainer struct {
		Name     string `json:"name"`
		Level    int    `json:"level"`
		XP       int    `json:"xp"`
		HP       int    `json:"hp"`
		MaxHP    int    `json:"maxHp"`
		Gold     int    `json:"gold"`
		Diamonds int    `json:"diamonds"`
	} `json:"trainer"`
}

// Login realiza a autenticação na API do jogo
func (c *APIClient) Login(email, password string) (*LoginResponse, error) {
	reqBody := LoginRequest{
		Email:    email,
		Password: password,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar login request: %w", err)
	}

	req, err := http.NewRequest("POST", BaseURL+"/auth/login", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição de login: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", OriginURL+"/")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha na requisição de login: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler resposta de login: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp LoginResponse
		_ = json.Unmarshal(respBytes, &errResp)
		if errResp.Error != "" {
			return nil, fmt.Errorf("login falhou (%d): %s (código: %s)", resp.StatusCode, errResp.Error, errResp.Code)
		}
		return nil, fmt.Errorf("login falhou com status %d: %s", resp.StatusCode, string(respBytes))
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(respBytes, &loginResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar JSON de login: %w", err)
	}

	if loginResp.AccessToken == "" {
		return nil, fmt.Errorf("token de acesso vazio retornado pelo servidor")
	}

	return &loginResp, nil
}

// GetTicket obtém o ticket para abrir o WebSocket do jogo
func (c *APIClient) GetTicket(accessToken string) (*TicketResponse, error) {
	req, err := http.NewRequest("POST", BaseURL+"/ws/ticket", bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição do ticket: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", OriginURL+"/")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha na requisição de ticket: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler resposta do ticket: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ticket falhou com status %d: %s", resp.StatusCode, string(respBytes))
	}

	var ticketResp TicketResponse
	if err := json.Unmarshal(respBytes, &ticketResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar ticket JSON: %w", err)
	}

	if ticketResp.Ticket == "" {
		return nil, fmt.Errorf("ticket vazio retornado pelo servidor")
	}

	return &ticketResp, nil
}

// GetMe obtém o perfil atual do treinador
func (c *APIClient) GetMe(accessToken string) (*TrainerInfo, error) {
	req, err := http.NewRequest("GET", BaseURL+"/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", OriginURL+"/")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("falha em /me (%d): %s", resp.StatusCode, string(body))
	}

	var info TrainerInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}
