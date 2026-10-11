# ⚡ Sigma Project Multi-Bot (Go Native Engine)

Bot autônomo, nativo e de alta performance desenvolvido em **Go (Golang)** para o jogo idle [Pokemon Sigma Project](https://poke.sigmaproject.online/play).

Inspirado na arquitetura do projeto `huntera-go`:
- **Ultra-leve**: Consome menos de ~15MB de RAM total e pouquíssimo uso de CPU.
- **Multi-contas nativo**: Cada conta roda em uma **Goroutine isolada** com reconexão automática e backoff exponencial.
- **Painel Web Cyber/Dark**: Dashboard em tempo real embutido (`http://localhost:3001`) com estatísticas de todas as contas, controle de rotas, cura e auto-hunt.
- **Monitor no Terminal**: Estatísticas de RAM, Goroutines, Kills, Capturas, Shinies, Vida e Ouro atualizadas a cada 4 segundos no console.

---

## 🚀 Como Executar

### 1. Configurar Contas
Abra o arquivo `accounts.json` e adicione suas contas:

```json
[
  {
    "id": "acc1",
    "name": "Treinador Ash",
    "email": "seu_email@gmail.com",
    "password": "sua_senha",
    "huntSlug": "caterpie",
    "autoHunt": true,
    "autoCatch": true,
    "ballId": 1,
    "shinyBallId": 5,
    "autoHeal": true,
    "autoDaily": true,
    "autoSpin": true,
    "enabled": true
  }
]
```

#### Opções de Configuração:
- `id`: Identificador único da conta (ex: `acc1`).
- `name`: Nome ou apelido de exibição.
- `email`: E-mail de login da conta no jogo.
- `password`: Senha da conta.
- `huntSlug`: Rota ou Pokémon alvo para caçar (ex: `caterpie`, `bellsprout`, `geodude`, `magikarp`, etc.). Se deixado em branco, mantém o mapa atual.
- `autoHunt`: Ativa o motor nativo de caça contínua do servidor (`true`/`false`).
- `autoCatch`: Tenta capturar Pokémons encontrados (`true`/`false`).
- `ballId`: ID da Pokébola padrão (1: Poke Ball, 2: Great Ball, 3: Super Ball, 4: Ultra Ball).
- `shinyBallId`: ID da Pokébola prioritária ao encontrar Shinies (5: Master Ball, 4: Ultra Ball).
- `autoHeal`: Retorna e cura no Centro Pokémon se a vida ficar baixa ou o líder desmaiar (`true`/`false`).
- `autoDaily`: Coleta login diário, recompensas de quests, bestiário e tarefas automaticamente (`true`/`false`).
- `autoSpin`: Gira a roda da sorte diária caso haja giros disponíveis (`true`/`false`).
- `enabled`: Habilita ou desabilita a execução daquela conta específica (`true`/`false`).

---

### 2. Rodar o Bot

No terminal:
```bash
./bot-sigma.exe
```
Ou com Go:
```bash
go run .
```

Para rodar apenas uma conta específica:
```bash
$env:ACCOUNT="acc1"; ./bot-sigma.exe
```

---

### 3. Painel de Controle (Web Dashboard)

Acesse no navegador:
👉 **[http://localhost:3002](http://localhost:3002)**
*(Na VPS: `http://SEU_IP_DA_VPS:3002`)*

Recursos do Dashboard:
- 📊 **Métricas Globais**: Total de Kills, Capturas, Shinies e Ouro acumulado de todas as contas.
- 🎴 **Cards Individuais**: Vida (HP), XP até o próximo nível, Pokémon Líder, Alvo atual e Ouro.
- ✈️ **Viajar / Trocar de Rota**: Dropdown interativo com todas as rotas e monstros do catálogo.
- 🏥 **Curar Agora**: Envia requisição de cura imediata no Centro Pokémon.
- 🎁 **Recompensas**: Coleta tarefas e bônus pendentes com um clique.
- ⚔️ **Toggle Auto-Hunt**: Pausa ou ativa o auto-hunt no servidor.
