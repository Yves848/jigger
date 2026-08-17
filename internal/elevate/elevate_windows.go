package elevate

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// ── Le `sudo` de Windows ──────────────────────────────────────────────────────────────
//
// Windows 11 livre `C:\WINDOWS\system32\sudo.exe` depuis la build 26052, mais **désactivé
// par défaut** : il faut l'allumer dans Paramètres → Système → Pour les développeurs.
// Mesuré sur la machine de développement (build 26200) :
//
//	> sudo config
//	Sudo est désactivé sur cet ordinateur. Pour l'activer, accédez à Developer Settings
//
// L'état se lit dans le registre plutôt qu'en lançant `sudo config` : un processus de
// moins, et une réponse qui ne dépend pas de la locale du message. La clé est **absente**
// tant que la fonction n'a jamais été activée — absente ou nulle valent désactivé.
//
// Le *mode* (nouvelle fenêtre, entrée désactivée, en ligne) n'est pas lu, et c'est
// délibéré : jigger n'a pas à promettre où la sortie s'affichera. Il annonce « sudo », pas
// « dans cette console ».
const (
	cleSudo    = `SOFTWARE\Microsoft\Windows\CurrentVersion\Sudo`
	valeurSudo = "Enabled"
)

func sudoActif() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, cleSudo, registry.QUERY_VALUE)
	if err != nil {
		return false // clé absente : jamais activé
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue(valeurSudo)
	if err != nil {
		return false
	}
	if v == 0 {
		return false
	}
	// Un `sudo` activé mais introuvable dans le PATH ne sert à rien. Le test vient en
	// second : c'est le registre qui dit si la fonction *du système* est utilisable, et
	// c'est la seule des deux choses que jigger sache vérifier. Un `sudo` tiers (gsudo,
	// celui de scoop) passe donc par la fenêtre élevée — qui marche tout autant, et dont
	// on connaît le comportement.
	_, err = exec.LookPath("sudo")
	return err == nil
}

// Prevue dit par quel chemin le rejeu passera.
func Prevue() Voie {
	if sudoActif() {
		return VoieSudo
	}
	return VoieFenetre
}

// Rejouer relance la commande avec les privilèges d'administrateur et rend son code de
// sortie. L'appelant a déjà obtenu un oui explicite.
func Rejouer(cmd string, argv []string) (int, error) {
	if sudoActif() {
		return parSudo(cmd, argv)
	}
	return parFenetre(cmd, argv)
}

// parSudo passe par le chemin ordinaire : un sous-processus qui hérite du terminal, comme
// tout ce que la façade relaie. C'est ce qui rend ce chemin-là agréable — rien de neuf ne
// s'interpose entre l'utilisateur et sa commande.
func parSudo(cmd string, argv []string) (int, error) {
	c := exec.Command("sudo", append([]string{cmd}, argv...)...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := c.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), nil // la commande a parlé : son code n'est pas notre erreur
	}
	if err != nil {
		return 1, err
	}
	return 0, nil
}

// ── ShellExecuteEx, verbe « runas » ───────────────────────────────────────────────────
//
// x/sys/windows expose `ShellExecute`, qui ne rend aucune poignée : impossible d'attendre
// le processus, donc impossible de rendre son code. La fenêtre se refermerait, et jigger
// n'aurait rien à dire. On déclare donc `ShellExecuteExW` — la variante qui remplit
// `hProcess` quand on le lui demande.

const (
	seeMaskNoCloseProcess = 0x00000040 // remplir hProcess, et ne pas le refermer
	seeMaskNoAsync        = 0x00000100 // ne pas rendre la main avant que l'appel soit fini
	swShowNormal          = 1
)

// shellExecuteInfo est SHELLEXECUTEINFOW. L'ordre des champs est celui de shellapi.h ; le
// bourrage entre `nShow` et `hInstApp`, puis entre `dwHotKey` et `hIcon`, est posé par Go
// exactement là où le compilateur C le pose — d'où `cbSize` calculé par unsafe.Sizeof
// plutôt qu'écrit à la main.
type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         windows.Handle
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     windows.Handle
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    windows.Handle
	dwHotKey     uint32
	hIcon        windows.Handle // union avec hMonitor
	hProcess     windows.Handle
}

var (
	shell32          = windows.NewLazySystemDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteExW")
)

func parFenetre(cmd string, argv []string) (int, error) {
	// ShellExecuteEx prend UNE ligne d'arguments, pas un tableau. EscapeArg applique les
	// règles de CommandLineToArgvW : sans elle, un identifiant winget à espaces
	// (« Microsoft.VisualStudio.2022.BuildTools » passe, mais pas tous) serait coupé en
	// deux arguments.
	ligne := ""
	for _, a := range argv {
		if ligne != "" {
			ligne += " "
		}
		ligne += syscall.EscapeArg(a)
	}

	verbe, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return 1, err
	}
	fichier, err := windows.UTF16PtrFromString(cmd)
	if err != nil {
		return 1, err
	}
	params, err := windows.UTF16PtrFromString(ligne)
	if err != nil {
		return 1, err
	}

	info := shellExecuteInfo{
		fMask:        seeMaskNoCloseProcess | seeMaskNoAsync,
		lpVerb:       verbe,
		lpFile:       fichier,
		lpParameters: params,
		nShow:        swShowNormal,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))

	r, _, errno := procShellExecute.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		// ERROR_CANCELLED : l'utilisateur a refusé l'invite UAC. Ce n'est pas une panne,
		// c'est une réponse — et l'appelant doit pouvoir la distinguer.
		if errno == windows.ERROR_CANCELLED {
			return 0, ErrRefuse
		}
		return 1, errno
	}
	defer windows.CloseHandle(info.hProcess)

	if _, err := windows.WaitForSingleObject(info.hProcess, windows.INFINITE); err != nil {
		return 1, err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(info.hProcess, &code); err != nil {
		return 1, err
	}
	return int(code), nil
}
