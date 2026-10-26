package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type MarketDiamondListing struct {
	ID         string `json:"id"`
	SellerName string `json:"sellerName"`
	Count      int    `json:"count"`
	Price      int    `json:"price"`
	Mode       string `json:"mode"`
	Currency   string `json:"currency"`
}

type MarketListingsResult struct {
	Listings []MarketDiamondListing `json:"listings"`
}

type CreateListingResponse struct {
	Listing struct {
		ID string `json:"id"`
	} `json:"listing"`
	Error string `json:"error,omitempty"`
}

// CreateMarketListing cria um anúncio no mercado (ex: para transferência de gold)
func (c *APIClient) CreateMarketListing(accessToken string, goods map[string]interface{}, price int, currency string) (string, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"goods":    goods,
		"price":    price,
		"currency": currency,
	})
	if err != nil {
		return "", fmt.Errorf("erro ao serializar listing payload: %w", err)
	}

	req, err := http.NewRequest("POST", BaseURL+"/market/listings", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisição de listing: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", OriginURL+"/")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("falha ao enviar anúncio ao mercado: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("erro ao criar anúncio (%d): %s", resp.StatusCode, string(body))
	}

	var res CreateListingResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta de criação de anúncio: %w", err)
	}

	if res.Listing.ID == "" {
		return "", fmt.Errorf("servidor não retornou ID para o anúncio: %s", string(body))
	}

	return res.Listing.ID, nil
}

// BuyMarketListing compra unidades de um anúncio no mercado
func (c *APIClient) BuyMarketListing(accessToken string, listingID string, count int) error {
	payload, err := json.Marshal(map[string]int{"count": count})
	if err != nil {
		return fmt.Errorf("erro ao serializar compra de anúncio: %w", err)
	}

	url := fmt.Sprintf("%s/market/listings/%s/buy", BaseURL, listingID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição de compra: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", OriginURL+"/")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao comprar anúncio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("falha na compra do anúncio (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteMarketListing cancela um anúncio aberto pelo jogador
func (c *APIClient) DeleteMarketListing(accessToken string, listingID string) error {
	url := fmt.Sprintf("%s/market/listings/%s", BaseURL, listingID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar requisição de cancelamento: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", OriginURL+"/")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao cancelar anúncio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("falha ao remover anúncio (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetDiamondListings obtém todas as ofertas de diamantes no mercado
func (c *APIClient) GetDiamondListings(accessToken string) ([]MarketDiamondListing, error) {
	req, err := http.NewRequest("GET", BaseURL+"/market/listings?kind=diamond", nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar busca de diamantes: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", OriginURL+"/")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar ofertas de diamantes: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro ao buscar diamantes no mercado (%d): %s", resp.StatusCode, string(body))
	}

	var res MarketListingsResult
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("erro ao decodificar JSON de diamantes: %w", err)
	}

	return res.Listings, nil
}

// BuyVipPackage compra um pacote VIP (ex: 30 dias) na loja oficial usando diamantes
func (c *APIClient) BuyVipPackage(accessToken string, days int) error {
	payload, err := json.Marshal(map[string]interface{}{
		"what": "vip",
		"days": days,
	})
	if err != nil {
		return fmt.Errorf("erro ao serializar compra de VIP: %w", err)
	}

	req, err := http.NewRequest("POST", BaseURL+"/purchases", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição de VIP: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", OriginURL+"/")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao enviar pedido de VIP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("falha na compra de VIP (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// ClaimAllMail resgata todas as cartas pendentes no correio do treinador
func (c *APIClient) ClaimAllMail(accessToken string) error {
	req, err := http.NewRequest("POST", BaseURL+"/mail/claim-all", bytes.NewReader([]byte("{}")))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição de correio: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", OriginURL+"/")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao resgatar correio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("falha ao resgatar correio (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}
