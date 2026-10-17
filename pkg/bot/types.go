package bot

// AccountConfig contém as configurações de uma conta individual
type AccountConfig struct {
	ID          string `json:"id"`                    // Identificador único (ex: "acc1")
	Name        string `json:"name"`                  // Nome amigável / Alias
	Email       string `json:"email"`                 // E-mail de login
	Password    string `json:"password"`              // Senha
	HuntSlug    string `json:"huntSlug,omitempty"`    // Rota/Monstro alvo para caçar (ex: "caterpie", "bellsprout")
	AutoHunt    bool   `json:"autoHunt"`              // Ativar auto-hunt no servidor (default: true)
	AutoCatch   bool   `json:"autoCatch"`             // Tentar capturar pokémons selvagens (default: true)
	BallID      int    `json:"ballId,omitempty"`      // Pokébola padrão (1: Poke Ball, 2: Great Ball, 3: Super, 4: Ultra)
	ShinyBallID int    `json:"shinyBallId,omitempty"` // Pokébola para Shiny (5: Master Ball, 4: Ultra Ball)
	AutoHeal    bool   `json:"autoHeal"`              // Curar no Centro Pokémon ao ficar com pouca vida (default: true)
	AutoDaily   bool   `json:"autoDaily"`             // Coletar login diário, recompensas de quests e tarefas (default: true)
	AutoSpin    bool   `json:"autoSpin"`              // Girar a roda da sorte quando houver giros (default: true)
	AutoNap     bool   `json:"autoNap,omitempty"`     // Gerenciar soneca/descanso automático (default: false)
	Strategy            string `json:"strategy,omitempty"`            // "rush" (padrão de XP) ou "bestiary" (tour completo de degraus)
	BestiaryTargetKills int    `json:"bestiaryTargetKills,omitempty"` // Meta de abates por monstro no modo bestiário (default: 120)
	BindIP              string `json:"bindIP,omitempty"`              // IP local para bind da conexão (ex: "10.0.0.191", "10.0.0.49")
	Enabled             bool   `json:"enabled"`                       // Se a conta está ativa para rodar (default: true)
}

// AccountStatus reflete o estado em tempo real da conta para exibição no Terminal e no Dashboard Web
type AccountStatus struct {
	ID                   string  `json:"id"`
	Alias                string  `json:"alias"`
	TrainerName          string  `json:"trainerName"`
	BindIP               string  `json:"bindIp,omitempty"`
	OutboundIP           string  `json:"outboundIp,omitempty"`
	Strategy             string  `json:"strategy,omitempty"`
	Level                int     `json:"level"`
	XP                   int     `json:"xp"`
	XPIntoLevel          int     `json:"xpIntoLevel"`
	XPToNextLevel        int     `json:"xpToNextLevel"`
	XPPercent            float64 `json:"xpPercent"`
	HP                   int     `json:"hp"`
	MaxHP                int     `json:"maxHp"`
	HPPercent            float64 `json:"hpPercent"`
	Gold                 int     `json:"gold"`
	Diamonds             int     `json:"diamonds"`
	InGame               bool    `json:"inGame"`
	InHunt               bool    `json:"inHunt"`
	MapSlug              string  `json:"mapSlug"`
	MapName              string  `json:"mapName"`
	LeaderSpeciesID      int     `json:"leaderSpeciesId,omitempty"`
	LeaderName           string  `json:"leaderName"`
	LeaderLevel          int     `json:"leaderLevel"`
	LeaderHP             int     `json:"leaderHp"`
	LeaderMaxHP          int     `json:"leaderMaxHp"`
	LeaderShiny          bool    `json:"leaderShiny"`
	WildTarget           string  `json:"wildTarget"`
	WildHP               int     `json:"wildHp"`
	WildMaxHP            int     `json:"wildMaxHp"`
	WildLevel            int     `json:"wildLevel"`
	WildShiny            bool    `json:"wildShiny"`
	AutoHuntActive       bool    `json:"autoHuntActive"`
	Kills                int     `json:"kills"`
	Captures             int     `json:"captures"`
	Shinies              int     `json:"shinies"`
	GoldFound            int     `json:"goldFound"`
	TrainerXPGained      int     `json:"trainerXpGained"`
	KillsPerMin          float64 `json:"killsPerMin"`
	XPPerHour            float64 `json:"xpPerHour"`
	BestiaryLevel          int     `json:"bestiaryLevel,omitempty"`
	BestiaryPoints         int     `json:"bestiaryPoints,omitempty"`
	BestiaryNextPoints     int     `json:"bestiaryNextPoints,omitempty"`
	BestiaryTargetKills    int     `json:"bestiaryTargetKills,omitempty"`
	BestiaryTotalKills     int     `json:"bestiaryTotalKills,omitempty"`
	BestiaryCurrentKills   int     `json:"bestiaryCurrentKills,omitempty"`
	BestiaryCompletedCount int     `json:"bestiaryCompletedCount,omitempty"`
	BestiaryPendingCount   int     `json:"bestiaryPendingCount,omitempty"`
	LastAction           string  `json:"lastAction"`
	LastError            string  `json:"lastError,omitempty"`
	ConnectedAt          int64   `json:"connectedAt"`
	UpdatedAt            int64   `json:"updatedAt"`
}

// InventoryItem representa um item de saque ou suprimento no inventário
type InventoryItem struct {
	ItemID    int    `json:"itemId"`
	Count     int    `json:"count"`
	Category  string `json:"category"`
	SellPrice int    `json:"sellPrice"`
	Locked    bool   `json:"locked"`
}

// Estruturas do Snapshot do Sigma Project recebido via WebSocket

type SnapshotMessage struct {
	Type     string         `json:"type,omitempty"`
	Tick     int64          `json:"tick,omitempty"`
	Online   int            `json:"online,omitempty"`
	Napping  int            `json:"napping,omitempty"`
	Content  string         `json:"content,omitempty"`
	Snapshot *GameStateData `json:"snapshot,omitempty"`
	Events   []EventData    `json:"events,omitempty"`
}

type EventData struct {
	Type      string `json:"type"`
	SpeciesID int    `json:"speciesId,omitempty"`
	WildID    int    `json:"wildId,omitempty"`
	Shiny     bool   `json:"shiny,omitempty"`
	Gold      int    `json:"gold,omitempty"`
	XP        int    `json:"xp,omitempty"`
	Level     int    `json:"level,omitempty"`
	BallID    int    `json:"ballId,omitempty"`
	Success   bool   `json:"success,omitempty"`
}

type GameStateData struct {
	Trainer    TrainerData        `json:"trainer"`
	Pokedex    PokedexData        `json:"pokedex"`
	Leader     *LeaderData        `json:"leader"`
	Hunt       *HuntData          `json:"hunt"`
	InCity     bool               `json:"inCity"`
	Bot        *BotPolicyData     `json:"bot"`
	AutoHunt   bool               `json:"autoHunt"`
	Balls      map[string]int     `json:"balls"`
	BoundBalls map[string]int     `json:"boundBalls"`
	Team       []TeamPokemonData  `json:"team"`
	Analyzer   *AnalyzerData      `json:"analyzer"`
	City       *CityData          `json:"city"`
	DailyLogin *DailyLoginData    `json:"dailyLogin"`
	Quests     []QuestData        `json:"quests"`
	Inventory  []InventoryItem    `json:"inventory"`
}

type TrainerData struct {
	Name           string `json:"name"`
	Level          int    `json:"level"`
	XP             int    `json:"xp"`
	XPIntoLevel    int    `json:"xpIntoLevel"`
	XPToNextLevel  int    `json:"xpToNextLevel"`
	HP             int    `json:"hp"`
	MaxHP          int    `json:"maxHp"`
	TotalKills     int    `json:"totalKills"`
	Gold           int    `json:"gold"`
	Diamonds       int    `json:"diamonds"`
	WheelSpins     map[string]int `json:"wheelSpins"`
}

type PokedexData struct {
	Seen   int `json:"seen"`
	Caught int `json:"caught"`
	Shiny  int `json:"shiny"`
	Total  int `json:"total"`
}

type LeaderData struct {
	PokemonID string `json:"pokemonId"`
	SpeciesID int    `json:"speciesId"`
	Level     int    `json:"level"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"maxHp"`
	Shiny     bool   `json:"shiny"`
}

type HuntData struct {
	MapSlug       string      `json:"mapSlug"`
	WildSpeciesID int         `json:"wildSpeciesId"`
	WildHP        int         `json:"wildHp"`
	WildMaxHP     int         `json:"wildMaxHp"`
	WildShiny     bool        `json:"wildShiny"`
	WildLevel     int         `json:"wildLevel"`
	Wilds         []WildData  `json:"wilds"`
	Engaged       bool        `json:"engaged"`
	TrainerStands bool        `json:"trainerStands"`
}

type WildData struct {
	ID        int    `json:"id"`
	SpeciesID int    `json:"speciesId"`
	Level     int    `json:"level"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"maxHp"`
	Shiny     bool   `json:"shiny"`
	Dead      bool   `json:"dead"`
	InRange   bool   `json:"inRange"`
}

type BotPolicyData struct {
	AutoCatch       bool  `json:"autoCatch"`
	BallID          *int  `json:"ballId"`
	ShinyBallID     *int  `json:"shinyBallId"`
	CaptureShiny    bool  `json:"captureShiny"`
	LureMin         int   `json:"lureMin"`
	FallbackAnyBall bool  `json:"fallbackAnyBall"`
}

type AnalyzerData struct {
	ElapsedMs     int64 `json:"elapsedMs"`
	Kills         int   `json:"kills"`
	TrainerXP     int   `json:"trainerXp"`
	Captures      int   `json:"captures"`
	ShinyKills    int   `json:"shinyKills"`
	ShinyCaptures int   `json:"shinyCaptures"`
	LootValue     int   `json:"lootValue"`
	GoldFound     int   `json:"goldFound"`
}

type CityData struct {
	MapSlug    string `json:"mapSlug"`
	PokeCenter *struct {
		X       int  `json:"x"`
		Y       int  `json:"y"`
		InRange bool `json:"inRange"`
	} `json:"pokeCenter"`
}

type DailyLoginData struct {
	Step       int  `json:"step"`
	Pending    int  `json:"pending"`
	CanAdvance bool `json:"canAdvance"`
}

type QuestData struct {
	Caught  int  `json:"caught"`
	Balls   int  `json:"balls"`
	Potions int  `json:"potions"`
	Done    bool `json:"done"`
	Claimed bool `json:"claimed"`
}

type TeamPokemonData struct {
	PokemonID string `json:"pokemonId"`
	SpeciesID int    `json:"speciesId"`
	Level     int    `json:"level"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"maxHp"`
	Shiny     bool   `json:"shiny"`
}

// Intenções enviadas ao servidor

type SetAutoHuntIntent struct {
	Type string `json:"type"` // "setAutoHunt"
	On   bool   `json:"on"`
}

type BotPolicyConfig struct {
	BallID          *int  `json:"ballId"`
	ShinyBallID     *int  `json:"shinyBallId"`
	CaptureShiny    bool  `json:"captureShiny"`
	LureMin         int   `json:"lureMin"`
	FallbackAnyBall bool  `json:"fallbackAnyBall"`
	Potions         []any `json:"potions"`
	AutoRevive      bool  `json:"autoRevive"`
	AutoElixir      bool  `json:"autoElixir,omitempty"`
}

type SetBotPolicyIntent struct {
	Type   string          `json:"type"` // "setBotPolicy"
	Policy BotPolicyConfig `json:"policy"`
}

type TravelIntent struct {
	Type    string `json:"type"` // "travel"
	MapSlug string `json:"mapSlug"`
}

type HealAtCenterIntent struct {
	Type string `json:"type"` // "healAtCenter"
}

type SimpleIntent struct {
	Type string `json:"type"`
}

type AttackIntent struct {
	Type   string `json:"type"` // "attack"
	WildID int    `json:"wildId"`
}

type ThrowBallIntent struct {
	Type   string `json:"type"` // "throwBall"
	BallID int    `json:"ballId"`
	WildID *int   `json:"wildId,omitempty"`
}

type SpinWheelIntent struct {
	Type    string `json:"type"` // "spinWheel"
	WheelID string `json:"wheelId"`
	Times   int    `json:"times"`
}
