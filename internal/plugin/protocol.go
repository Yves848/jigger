package plugin

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// delaiMax borne le temps qu'un plugin peut prendre pour répondre. Le réchauffement
// tourne hors du chemin du rendu, donc l'attente est tolérable ; ce qui ne l'est pas,
// c'est un plugin qui ne rend jamais la main et laisse `jigger warm` planté derrière
// son verrou.
const delaiMax = 30 * time.Second

// Run lance un binaire de plugin, capture sa sortie standard et la rend. Elle sert au
// réchauffement des caches — l'exécution d'un verbe, elle, passe par la façade, qui relaie
// le terminal (cf. facade.ExecuterAvec).
//
// L'erreur est **rendue**, jamais avalée : un plugin qui échoue doit laisser le cache
// précédent en place plutôt que de l'écraser par du vide. Le stderr du plugin est repris
// dans le message, car c'est là qu'il explique ce qui lui manque.
func Run(binaire string, args []string) ([]byte, error) {
	return runDelai(binaire, args, delaiMax)
}

// delaiVivierDefaut borne un vivier « direct », qui tourne DANS le chemin du rendu. Il est
// court par construction : au-delà, le prompt attendrait, et un popup en retard est pire
// qu'un popup vide. La mesure qui a fixé l'ordre de grandeur est dans l'ADR-0009 — une
// frappe coûte 11 ms, une réponse git contextuelle 7.
const delaiVivierDefaut = 200 * time.Millisecond

// delaiVivier rend le délai d'un vivier direct. JIGGER_DELAI_VIVIER, en millisecondes,
// permet de le resserrer sur une machine lente à diagnostiquer — et aux tests de ne pas
// durer deux cents millisecondes chacun.
func delaiVivier() time.Duration {
	if v := os.Getenv("JIGGER_DELAI_VIVIER"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return delaiVivierDefaut
}

func runDelai(binaire string, args []string, delai time.Duration) ([]byte, error) {
	c := exec.Command(binaire, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr

	if err := c.Start(); err != nil {
		return nil, err
	}

	fini := make(chan error, 1)
	go func() { fini <- c.Wait() }()

	select {
	case err := <-fini:
		if err != nil {
			return nil, echec(binaire, err, stderr.Bytes())
		}
		return stdout.Bytes(), nil
	case <-time.After(delai):
		_ = c.Process.Kill()
		// Le processus est réclamé EN ARRIÈRE-PLAN, et non attendu ici. Attendre sur
		// place rendait le délai inopérant : `c.Wait()` ne rend la main qu'une fois le
		// tuyau de sortie fermé, or un plugin qui est lui-même un script laisse ses
		// enfants le tenir ouvert après sa mort — un `sleep` a fait attendre cinq
		// secondes un délai réglé sur cinquante millisecondes.
		//
		// La goroutine ne fuit pas plus que le processus qu'elle attend : elle se termine
		// avec lui. On garde ainsi la promesse de l'ADR-0009 — le prompt n'attend pas —
		// sans laisser de zombie derrière soi.
		go func() { <-fini }()
		return nil, fmt.Errorf("%s : pas de réponse après %s", binaire, delai)
	}
}

// echec compose un message d'erreur qui dit ce que le plugin a répondu. Sans lui, un
// « exit status 1 » laisserait l'utilisateur sans la moindre piste.
func echec(binaire string, err error, stderr []byte) error {
	msg := strings.TrimSpace(string(stderr))
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if msg == "" {
			return fmt.Errorf("%s : code de sortie %d", binaire, ee.ExitCode())
		}
		return fmt.Errorf("%s : code de sortie %d — %s", binaire, ee.ExitCode(), msg)
	}
	if msg == "" {
		return fmt.Errorf("%s : %w", binaire, err)
	}
	return fmt.Errorf("%s : %w — %s", binaire, err, msg)
}
