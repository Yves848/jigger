//go:build !windows

package pm

import (
	"os/exec"
	"syscall"
)

// detach détache le fils du terminal de contrôle : un `jigger warm` lancé depuis un
// widget ne doit ni recevoir le ^C de l'utilisateur, ni mourir avec le shell.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// hide n'a de sens que sous Windows (aucune fenêtre à cacher ailleurs).
func hide(*exec.Cmd) {}
