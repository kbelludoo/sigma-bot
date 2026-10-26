package pool

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"bot-sigma/pkg/bot"
	"bot-sigma/pkg/client"
)

type BotProvider interface {
	GetConfig() bot.AccountConfig
	GetStatus() bot.AccountStatus
	GetAPIClient() *client.APIClient
	GetAccessToken() string
}

type Coordinator struct {
	mu           sync.Mutex
	bots         []BotProvider
	vipPriority  []string // IDs das contas na ordem de prioridade para VIP
	goldReserve  int      // Reserva de ouro que a conta doadora não doa (ex: 2.000.000)
	pollInterval time.Duration
	running      bool
}

func NewCoordinator(bots []BotProvider, priority []string) *Coordinator {
	if len(priority) == 0 {
		priority = []string{"acc3", "acc2", "acc1"}
	}
	return &Coordinator{
		bots:         bots,
		vipPriority:  priority,
		goldReserve:  2_000_000, // 2M de reserva para Pokébolas e suprimentos
		pollInterval: 60 * time.Second,
	}
}

// Start inicia o loop em segundo plano do coordenador de Gold Pooling e Auto-VIP
func (c *Coordinator) Start(ctx context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	log.Println("💰 [GOLD POOLING] Coordenador de Pooling & Auto-VIP iniciado!")

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("💰 [GOLD POOLING] Coordenador encerrado com sucesso.")
			return
		case <-ticker.C:
			c.evaluatePool(ctx)
		}
	}
}

// evaluatePool avalia saldos e orquestra compras ou repasses
func (c *Coordinator) evaluatePool(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Mapear bots por ID
	botMap := make(map[string]BotProvider)
	for _, b := range c.bots {
		cfg := b.GetConfig()
		botMap[cfg.ID] = b
	}

	// 2. Identificar a próxima conta na fila de prioridade que ainda NÃO tem VIP
	var targetBot BotProvider
	var targetID string

	for _, id := range c.vipPriority {
		b, ok := botMap[id]
		if !ok {
			continue
		}
		st := b.GetStatus()
		if !st.InGame {
			continue // Aguardar estar online
		}
		if !st.VipActive {
			targetBot = b
			targetID = id
			break
		}
	}

	// Se todas as contas já tiverem VIP ativa, não há necessidade de repasse emergencial
	if targetBot == nil {
		return
	}

	targetToken := targetBot.GetAccessToken()
	targetAPI := targetBot.GetAPIClient()
	if targetToken == "" || targetAPI == nil {
		return
	}

	targetStatus := targetBot.GetStatus()
	targetName := targetStatus.TrainerName
	if targetName == "" {
		targetName = targetStatus.Alias
	}

	// 3. Caso a conta alvo já possua 20 diamantes: ativa direto!
	if targetStatus.Diamonds >= 20 {
		log.Printf("💎 [AUTO-VIP] %s (%s) possui %d diamantes (>= 20)! Ativando 30 dias de VIP...",
			targetName, targetID, targetStatus.Diamonds)
		c.executeVipActivation(targetAPI, targetToken, targetName)
		return
	}

	// 4. Consultar o mercado para obter o custo de compra dos diamantes faltantes
	neededDiamonds := 20 - targetStatus.Diamonds
	listings, err := targetAPI.GetDiamondListings(targetToken)
	if err != nil || len(listings) == 0 {
		return
	}

	sort.Slice(listings, func(i, j int) bool {
		return listings[i].Price < listings[j].Price
	})

	costInGold := 0
	accumulated := 0
	for _, l := range listings {
		if l.Mode != "fixed" || l.Price <= 0 {
			continue
		}
		toTake := neededDiamonds - accumulated
		if toTake <= 0 {
			break
		}
		count := l.Count
		if count > toTake {
			count = toTake
		}
		costInGold += count * l.Price
		accumulated += count
	}

	if accumulated < neededDiamonds {
		return // Poucas ofertas no mercado no momento
	}

	// 5. Se a própria conta alvo já possui ouro suficiente: compra sozinha!
	if targetStatus.Gold >= costInGold {
		log.Printf("💰 [AUTO-VIP] %s (%s) possui ouro suficiente (%s gp >= %s gp)! Comprando diamantes...",
			targetName, targetID, formatGold(targetStatus.Gold), formatGold(costInGold))
		c.buyDiamondsAndActivateVip(targetAPI, targetToken, targetName, neededDiamonds)
		return
	}

	// 6. Caso precise de repasse (Gold Pooling)
	neededNetGold := costInGold - targetStatus.Gold
	// Taxa de 5% da casa: gross = net / 0.95
	neededGrossGold := int(float64(neededNetGold)/0.95) + 50_000

	// Localizar contas doadoras com VIP já ativa e saldo acima da reserva
	var donorBot BotProvider

	for id, b := range botMap {
		if id == targetID {
			continue
		}
		st := b.GetStatus()
		if !st.InGame || !st.VipActive {
			continue
		}
		avail := st.Gold - c.goldReserve
		if avail >= neededGrossGold {
			donorBot = b
			break
		}
	}

	if donorBot == nil {
		// Nenhuma doadora única cobre o valor total ainda
		return
	}

	donorStatus := donorBot.GetStatus()
	donorName := donorStatus.TrainerName
	if donorName == "" {
		donorName = donorStatus.Alias
	}

	log.Printf("🤝 [GOLD POOLING] Repasse viável detectado! %s doará %s gp para %s ativar a VIP!",
		donorName, formatGold(neededGrossGold), targetName)

	// 7. Executar a transferência segura via Mercado P2P
	c.transferGoldViaMarket(ctx, targetBot, donorBot, neededGrossGold, targetName, donorName)
}

// transferGoldViaMarket executa a listagem e compra do repasse
func (c *Coordinator) transferGoldViaMarket(ctx context.Context, targetBot, donorBot BotProvider, amount int, targetName, donorName string) {
	targetToken := targetBot.GetAccessToken()
	targetAPI := targetBot.GetAPIClient()
	donorToken := donorBot.GetAccessToken()
	donorAPI := donorBot.GetAPIClient()

	// 1. Obter um Pokémon do Box do destinatário para listar
	storage, err := targetAPI.GetStorage(targetToken)
	if err != nil || len(storage.Box) == 0 {
		log.Printf("⚠️ [GOLD POOLING] %s não possui Pokémons no depósito para criar o anúncio.", targetName)
		return
	}

	// Escolher o primeiro Pokémon não travado
	var pokeID string
	for _, p := range storage.Box {
		if !p.Locked {
			pokeID = p.PokemonID
			break
		}
	}
	if pokeID == "" {
		pokeID = storage.Box[0].PokemonID
	}

	goods := map[string]interface{}{
		"kind":      "pokemon",
		"pokemonId": pokeID,
	}

	log.Printf("📦 [GOLD POOLING] %s criando anúncio no Mercado por %s gp...", targetName, formatGold(amount))
	listingID, err := targetAPI.CreateMarketListing(targetToken, goods, amount, "gold")
	if err != nil {
		log.Printf("❌ [GOLD POOLING] Falha ao criar anúncio: %v", err)
		return
	}

	time.Sleep(2 * time.Second)

	log.Printf("🛒 [GOLD POOLING] %s comprando o anúncio (%s)...", donorName, listingID)
	err = donorAPI.BuyMarketListing(donorToken, listingID, 1)
	if err != nil {
		log.Printf("❌ [GOLD POOLING] Falha na compra da doadora: %v. Cancelando anúncio...", err)
		_ = targetAPI.DeleteMarketListing(targetToken, listingID)
		return
	}

	log.Printf("✅ [GOLD POOLING] Transferência de %s gp concluída com sucesso!", formatGold(amount))
	time.Sleep(3 * time.Second)

	// Com o ouro recebido, a conta alvo compra os diamantes e ativa a VIP!
	c.buyDiamondsAndActivateVip(targetAPI, targetToken, targetName, 20)
}

func (c *Coordinator) buyDiamondsAndActivateVip(api *client.APIClient, token, name string, countNeeded int) {
	listings, err := api.GetDiamondListings(token)
	if err != nil {
		log.Printf("❌ [AUTO-VIP] Falha ao consultar diamantes no mercado: %v", err)
		return
	}

	sort.Slice(listings, func(i, j int) bool {
		return listings[i].Price < listings[j].Price
	})

	bought := 0
	for _, l := range listings {
		if l.Mode != "fixed" || l.Price <= 0 {
			continue
		}
		toTake := countNeeded - bought
		if toTake <= 0 {
			break
		}
		buyCount := l.Count
		if buyCount > toTake {
			buyCount = toTake
		}

		err := api.BuyMarketListing(token, l.ID, buyCount)
		if err == nil {
			bought += buyCount
			log.Printf("💎 [AUTO-VIP] %s comprou %d diamante(s) de '%s'!", name, buyCount, l.SellerName)
		}
		time.Sleep(1 * time.Second)
	}

	time.Sleep(2 * time.Second)
	c.executeVipActivation(api, token, name)
}

func (c *Coordinator) executeVipActivation(api *client.APIClient, token, name string) {
	err := api.BuyVipPackage(token, 30)
	if err != nil {
		log.Printf("❌ [AUTO-VIP] Falha ao enviar pedido de VIP para %s: %v", name, err)
		return
	}

	time.Sleep(1500 * time.Millisecond)
	err = api.ClaimAllMail(token)
	if err != nil {
		log.Printf("⚠️ [AUTO-VIP] Pedido de VIP aceito, mas erro ao resgatar correio de %s: %v", name, err)
		return
	}

	log.Printf("🎉🎉🎉 [AUTO-VIP] SUCESSO ABSOLUTO! VIP DE 30 DIAS ATIVADA PARA %s! (+30%% EXP ATIVO) 🎉🎉🎉", name)
}

func formatGold(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	}
	return fmt.Sprintf("%.2fkk", float64(n)/1_000_000.0)
}
