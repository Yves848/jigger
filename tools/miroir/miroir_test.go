package main

import (
	"strings"
	"testing"
	"time"
)

func etat(tete string, tags map[string]string) Etat {
	return Etat{Tete: tete, Tags: tags}
}

func TestDeuxDepotsIdentiquesNeDonnentAucunEcart(t *testing.T) {
	a := etat("abc1234", map[string]string{"v0.12.0": "aaa", "v0.11.0": "bbb"})
	b := etat("abc1234", map[string]string{"v0.11.0": "bbb", "v0.12.0": "aaa"})

	if ecarts := Comparer(a, b); len(ecarts) != 0 {
		t.Fatalf("aucun écart attendu, obtenu %v", ecarts)
	}
}

func TestTeteDivergenteEstSignalee(t *testing.T) {
	a := etat("neuf", nil)
	b := etat("vieux", nil)

	ecarts := Comparer(a, b)
	if len(ecarts) != 1 {
		t.Fatalf("un écart attendu, obtenu %v", ecarts)
	}
	if ecarts[0].Quoi != "main" || ecarts[0].GitLab != "neuf" || ecarts[0].GitHub != "vieux" {
		t.Fatalf("écart mal formé : %+v", ecarts[0])
	}
}

// Le cas qui a réellement mordu : le miroir figé garde une tête ancienne et n'a jamais reçu
// les tags des versions publiées depuis.
func TestTagAbsentDuMiroirEstSignale(t *testing.T) {
	a := etat("meme", map[string]string{"v0.11.0": "bbb", "v0.12.0": "aaa"})
	b := etat("meme", map[string]string{"v0.11.0": "bbb"})

	ecarts := Comparer(a, b)
	if len(ecarts) != 1 {
		t.Fatalf("un écart attendu, obtenu %v", ecarts)
	}
	if ecarts[0].Quoi != "tag v0.12.0" || ecarts[0].GitHub != "" {
		t.Fatalf("écart mal formé : %+v", ecarts[0])
	}
}

// Un tag que seul le miroir porte compte aussi : le miroir est en mode push sans
// « keep divergent refs », donc GitHub ne devrait jamais rien avoir en propre.
func TestTagPresentSeulementSurLeMiroirEstSignale(t *testing.T) {
	a := etat("meme", nil)
	b := etat("meme", map[string]string{"v9.9.9": "zzz"})

	ecarts := Comparer(a, b)
	if len(ecarts) != 1 {
		t.Fatalf("un écart attendu, obtenu %v", ecarts)
	}
	if ecarts[0].Quoi != "tag v9.9.9" || ecarts[0].GitLab != "" {
		t.Fatalf("écart mal formé : %+v", ecarts[0])
	}
}

func TestTagDeplaceEstSignale(t *testing.T) {
	a := etat("meme", map[string]string{"v0.12.0": "refait"})
	b := etat("meme", map[string]string{"v0.12.0": "ancien"})

	ecarts := Comparer(a, b)
	if len(ecarts) != 1 || ecarts[0].Quoi != "tag v0.12.0" {
		t.Fatalf("un écart de tag attendu, obtenu %v", ecarts)
	}
}

// L'ordre est stable : le message d'une issue rouverte trois jours plus tard doit pouvoir
// se comparer à l'œil au précédent.
func TestLOrdreEstStable(t *testing.T) {
	a := etat("neuf", map[string]string{"v0.12.0": "a", "v0.10.0": "c", "v0.11.0": "b"})
	b := etat("vieux", nil)

	ecarts := Comparer(a, b)
	attendu := []string{"main", "tag v0.10.0", "tag v0.11.0", "tag v0.12.0"}
	if len(ecarts) != len(attendu) {
		t.Fatalf("%d écarts attendus, obtenu %v", len(attendu), ecarts)
	}
	for i, quoi := range attendu {
		if ecarts[i].Quoi != quoi {
			t.Fatalf("écart %d : %q attendu, obtenu %q", i, quoi, ecarts[i].Quoi)
		}
	}
}

func TestResumeDitCeQuiManqueEtOuIlManque(t *testing.T) {
	ecarts := []Ecart{
		{Quoi: "main", GitLab: "fadaf2d", GitHub: "7974687"},
		{Quoi: "tag v0.12.0", GitLab: "2c32bd1", GitHub: ""},
	}

	r := Resume(ecarts)
	for _, attendu := range []string{"main", "fadaf2d", "7974687", "tag v0.12.0", "absent"} {
		if !strings.Contains(r, attendu) {
			t.Fatalf("le résumé ne mentionne pas %q :\n%s", attendu, r)
		}
	}
}

// ── les jetons du projet ───────────────────────────────────────────────

func jeton(nom, expire string) Jeton {
	return Jeton{Nom: nom, ExpireLe: expire, Actif: true}
}

// Le 6 septembre 2026, pour que les tests ne dépendent pas du jour où on les lance.
func aujourdhui() time.Time { return time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC) }

func TestUnJetonLointainNAlarmePas(t *testing.T) {
	alertes := JetonsAlarmants([]Jeton{jeton("ci", "2026-12-31")}, aujourdhui(), 30)
	if len(alertes) != 0 {
		t.Fatalf("aucune alerte attendue à 116 jours, obtenu %v", alertes)
	}
}

func TestUnJetonProcheAlarme(t *testing.T) {
	// Le cas réel : garde-fou-miroir expirait dans dix jours sans que personne le sache.
	alertes := JetonsAlarmants([]Jeton{jeton("garde-fou", "2026-09-16")}, aujourdhui(), 30)
	if len(alertes) != 1 {
		t.Fatalf("une alerte attendue, obtenu %v", alertes)
	}
	if alertes[0].Jours != 10 {
		t.Errorf("Jours = %d, attendu 10", alertes[0].Jours)
	}
	if !strings.Contains(alertes[0].Raison, "10 jour") {
		t.Errorf("Raison = %q, doit dire le nombre de jours", alertes[0].Raison)
	}
}

func TestUnJetonExpireAlarmeAussi(t *testing.T) {
	// Passé l'échéance, l'alerte doit rester : c'est le moment où la panne est réelle.
	alertes := JetonsAlarmants([]Jeton{jeton("mort", "2026-09-01")}, aujourdhui(), 30)
	if len(alertes) != 1 || alertes[0].Jours >= 0 {
		t.Fatalf("alerte attendue avec un compte négatif, obtenu %v", alertes)
	}
	if !strings.Contains(alertes[0].Raison, "expiré") {
		t.Errorf("Raison = %q, doit dire qu'il est expiré", alertes[0].Raison)
	}
}

func TestUnJetonSansEcheanceNAlarmePas(t *testing.T) {
	// GitLab autorise des jetons sans date de fin : rien à surveiller.
	alertes := JetonsAlarmants([]Jeton{jeton("eternel", "")}, aujourdhui(), 30)
	if len(alertes) != 0 {
		t.Fatalf("aucune alerte attendue sans échéance, obtenu %v", alertes)
	}
}

func TestUnJetonInactifOuRevoqueEstIgnore(t *testing.T) {
	// Un jeton déjà révoqué n'est pas une panne à venir : il ne sert plus.
	js := []Jeton{
		{Nom: "revoque", ExpireLe: "2026-09-10", Actif: true, Revoque: true},
		{Nom: "inactif", ExpireLe: "2026-09-10", Actif: false},
	}
	if alertes := JetonsAlarmants(js, aujourdhui(), 30); len(alertes) != 0 {
		t.Fatalf("aucune alerte attendue, obtenu %v", alertes)
	}
}

func TestLesAlertesSontTrieesParUrgence(t *testing.T) {
	js := []Jeton{jeton("dans-20", "2026-09-26"), jeton("dans-3", "2026-09-09")}
	alertes := JetonsAlarmants(js, aujourdhui(), 30)
	if len(alertes) != 2 || alertes[0].Nom != "dans-3" {
		t.Fatalf("le plus urgent doit venir en tête, obtenu %v", alertes)
	}
}
