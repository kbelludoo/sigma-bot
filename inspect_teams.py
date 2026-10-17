import json
import urllib.request

with open('/home/ubuntu/sigma-bot/accounts.json') as f:
    accs = json.load(f)

# Catalog species map

species_map = {
    1: "Bulbasaur", 2: "Ivysaur", 3: "Venusaur",
    4: "Charmander", 5: "Charmeleon", 6: "Charizard",
    7: "Squirtle", 8: "Wartortle", 9: "Blastoise",
    10: "Caterpie", 13: "Weedle", 16: "Pidgey", 19: "Rattata",
    25: "Pikachu", 58: "Growlithe", 59: "Arcanine",
    60: "Poliwag", 61: "Poliwhirl", 62: "Poliwrath",
    74: "Geodude", 75: "Graveler", 76: "Golem",
    102: "Exeggcute", 103: "Exeggutor", 125: "Electabuzz",
    129: "Magikarp", 130: "Gyarados", 131: "Lapras"
}

for acc in accs:
    login_data = json.dumps({"email": acc["email"], "password": acc["password"]}).encode()
    req = urllib.request.Request(
        "https://api.poke.sigmaproject.online/auth/login",
        data=login_data,
        headers={"Content-Type": "application/json", "User-Agent": "Mozilla/5.0"}
    )
    res = json.loads(urllib.request.urlopen(req).read().decode())
    token = res["accessToken"]
    
    req_me = urllib.request.Request(
        "https://api.poke.sigmaproject.online/me",
        headers={"Authorization": f"Bearer {token}", "User-Agent": "Mozilla/5.0"}
    )
    me = json.loads(urllib.request.urlopen(req_me).read().decode())
    team = me.get("team", [])
    
    req_storage = urllib.request.Request(
        "https://api.poke.sigmaproject.online/storage",
        headers={"Authorization": f"Bearer {token}", "User-Agent": "Mozilla/5.0"}
    )
    try:
        storage = json.loads(urllib.request.urlopen(req_storage).read().decode())
    except Exception as e:
        storage = []
        
    print(f"\n==========================================")
    print(f"Conta: {acc['id']} - Treinador: {me['trainer']['name']} (Nv. {me['trainer']['level']})")
    print(f"Ouro: {me['trainer']['gold']} | Diamonds: {me['trainer']['diamonds']}")
    print(f"TEAM ({len(team)} Pokemons):")
    for p in team:
        s_id = p.get("speciesId")
        name = species_map.get(s_id, f"Species #{s_id}")
        slot = p.get("slot")
        print(f"  [Slot {slot}] {name} (ID: {p.get('pokemonId')}) | Lv: {p.get('level')} | HP: {p.get('hp')}/{p.get('maxHp')} | Shiny: {p.get('shiny')}")
    
    print(f"STORAGE response type: {type(storage)}")
    if isinstance(storage, dict):
        print(f"STORAGE keys: {storage.keys()}")
        pokemons = storage.get("pokemons") or storage.get("boxes") or []
        print(f"STORAGE pokemons count: {len(pokemons)}")
        for p in pokemons:
            if isinstance(p, dict):
                s_id = p.get("speciesId")
                name = species_map.get(s_id, f"Species #{s_id}")
                print(f"  - {name} | Lv: {p.get('level')} | Shiny: {p.get('shiny')} | ID: {p.get('pokemonId')}")
    elif isinstance(storage, list):
        print(f"STORAGE list count: {len(storage)}, sample: {storage[:3]}")

