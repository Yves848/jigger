package plugin

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
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
	case <-time.After(delaiMax):
		// Le processus est tué, puis attendu : sans ce Wait, on laisserait un zombie
		// derrière chaque plugin trop lent.
		_ = c.Process.Kill()
		<-fini
		return nil, fmt.Errorf("%s : pas de réponse après %s", binaire, delaiMax)
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
