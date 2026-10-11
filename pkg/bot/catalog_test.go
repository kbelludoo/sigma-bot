package bot

import (
	"testing"
)

func TestOptimalHuntTactical(t *testing.T) {
	cat := GetGlobalCatalog()

	// Bulbasaur (ID 1, Grass/Poison) no nível 23
	huntBulba, reasonBulba := cat.GetOptimalHuntForLeader(23, 1)
	t.Logf("Bulbasaur (Nv 23): Hunt='%s', Motivo='%s'", huntBulba, reasonBulba)

	// Charmander (ID 4, Fire) no nível 21
	huntChar, reasonChar := cat.GetOptimalHuntForLeader(21, 4)
	t.Logf("Charmander (Nv 21): Hunt='%s', Motivo='%s'", huntChar, reasonChar)

	// Bulbasaur no nível 30
	huntBulba30, reasonBulba30 := cat.GetOptimalHuntForLeader(30, 1)
	t.Logf("Bulbasaur (Nv 30): Hunt='%s', Motivo='%s'", huntBulba30, reasonBulba30)

	// Charmander no nível 30
	huntChar30, reasonChar30 := cat.GetOptimalHuntForLeader(30, 4)
	t.Logf("Charmander (Nv 30): Hunt='%s', Motivo='%s'", huntChar30, reasonChar30)

	// Squirtle (ID 7, Water) no nível 21
	huntSquirt, reasonSquirt := cat.GetOptimalHuntForLeader(21, 7)
	t.Logf("Squirtle (Nv 21): Hunt='%s', Motivo='%s'", huntSquirt, reasonSquirt)

	if huntBulba == "" || huntChar == "" {
		t.Fatalf("Rotas vazias retornadas!")
	}
}
