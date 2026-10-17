import json
import urllib.request

with open('/home/ubuntu/sigma-bot/accounts.json') as f:
    accs = json.load(f)

species_names = {
    1: "Bulbasaur", 2: "Ivysaur", 3: "Venusaur",
    4: "Charmander", 5: "Charmeleon", 6: "Charizard",
    7: "Squirtle", 8: "Wartortle", 9: "Blastoise",
    60: "Poliwag", 61: "Poliwhirl", 62: "Poliwrath",
    129: "Magikarp", 130: "Gyarados", 131: "Lapras"
}

for acc in accs:
    login_data = json.dumps({"email": acc["email"], "password": acc["password"]}).encode()
    req = urllib.request.Request(
        "https://api.poke.sigmaproject.online/auth/login",
        data=login_data,
        headers={"Content-Type": "application/json", "User-Agent": "Mozilla/5.0"}
    )
    token = json.loads(urllib.request.urlopen(req).read().decode())["accessToken"]

    req_storage = urllib.request.Request(
        "https://api.poke.sigmaproject.online/storage",
        headers={"Authorization": f"Bearer {token}", "User-Agent": "Mozilla/5.0"}
    )
    st = json.loads(urllib.request.urlopen(req_storage).read().decode())
    box = st.get("box", [])
    
    # Also get team
    req_me = urllib.request.Request(
        "https://api.poke.sigmaproject.online/me",
        headers={"Authorization": f"Bearer {token}", "User-Agent": "Mozilla/5.0"}
    )
    me = json.loads(urllib.request.urlopen(req_me).read().decode())
    team = me.get("team", [])
    
    print(f"\n========================================")
    print(f"Treinador: {acc['name']} (Nv. {me['trainer']['level']})")
    print(f"Time Ativo:")
    for p in team:
        name = species_names.get(p.get("speciesId"), f"Species #{p.get('speciesId')}")
        print(f"  * {name} (ID: {p.get('pokemonId')}) - Nv. {p.get('level')} | HP: {p.get('hp')}/{p.get('maxHp')}")
    
    print(f"Pokemons de Água / Starters no Box ({len(box)} no box):")
    water_or_starters = []
    for p in box:
        sid = p.get("speciesId")
        if sid in species_names or sid in [7, 8, 9, 60, 61, 62, 129, 130, 131]:
            name = species_names.get(sid, f"Species #{sid}")
            water_or_starters.append(f"{name} (Nv. {p.get('level')}, ID: {p.get('pokemonId')})")
    if water_or_starters:
        for w in water_or_starters:
            print("  -", w)
    else:
        print("  (Nenhum inicial de água encontrado no box)")
