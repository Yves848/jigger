package pm

import (
	"os/exec"
	"syscall"
)

// Drapeaux de CreateProcess. DETACHED_PROCESS coupe le fils de la console du shell —
// il ne peut donc plus rien écrire par-dessus le prompt, ni survivre en tenant un tuyau
// vers lui. CREATE_NO_WINDOW, lui, garde la console du parent mais n'en ouvre aucune :
// c'est ce qu'il faut pour un `winget` dont on lit la sortie.
const (
	detachedProcess = 0x00000008
	createNoWindow  = 0x08000000
)

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: detachedProcess}
}

func hide(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
