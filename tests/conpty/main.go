//go:build windows

// Harnais de test du module PowerShell : lance pwsh dans un vrai pseudo-terminal
// (ConPTY), tape une séquence de touches, et rend le flux que le shell a écrit.
//
// C'est le pendant Windows de tests/zpty.zsh, et il existe pour la même raison : le
// popup vivant ne se teste pas autrement. Tout ce qui compte — le cadre s'affiche, il
// disparaît, la ligne est réécrite — ne se voit que dans ce que PSReadLine écrit sur le
// terminal, et PSReadLine ne s'anime que devant une vraie console.
//
//	go run ./tests/conpty -rc setup.ps1 -keys 'winget ins\t\r'
//	go run ./tests/conpty -rc setup.ps1 -keys 'winget u' -visible
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func main() {
	rc := flag.String("rc", "", "script PowerShell chargé avant de taper (comme un ~/.zshrc)")
	keys := flag.String("keys", "", `touches à taper ; échappements Go acceptés (\t, \r, \x1b[B…)`)
	cmd := flag.String("cmd", `pwsh.exe -NoLogo -NoProfile -NoExit`, "programme lancé dans le pseudo-terminal")
	cols := flag.Int("cols", 120, "largeur du terminal")
	rows := flag.Int("rows", 30, "hauteur du terminal")
	settle := flag.Duration("settle", 2500*time.Millisecond, "attente au démarrage du shell")
	pause := flag.Duration("pause", 220*time.Millisecond, "attente après chaque touche")
	last := flag.Duration("last", 1500*time.Millisecond, "attente après la dernière touche")
	visible := flag.Bool("visible", false, "retirer les séquences ANSI")
	ecran := flag.Bool("screen", false, "rendre l'écran final, tel qu'on le verrait")
	flag.Parse()

	touches, err := unescape(*keys)
	if err != nil {
		fmt.Fprintln(os.Stderr, "touches illisibles :", err)
		os.Exit(2)
	}

	out, err := run(*rc, touches, *cmd, *cols, *rows, *settle, *pause, *last)
	if err != nil {
		fmt.Fprintln(os.Stderr, "harnais :", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "harnais : %d octets capturés\n", len(out))
	switch {
	case *ecran:
		e := NewEcran(*cols, *rows)
		e.Ecrire(out)
		ligne, colonne := e.Curseur()
		fmt.Printf("%s[curseur : ligne %d, colonne %d]\n", e.String(), ligne, colonne)
	case *visible:
		fmt.Print(strip(out))
	default:
		fmt.Print(out)
	}
}

// run monte le pseudo-terminal, y lance le shell, tape, et rend tout ce qu'il a écrit.
//
// Les tuyaux sont créés à la main plutôt qu'avec os.Pipe : celui de Go passe par le
// scrutateur d'E/S du runtime, et ConPTY, qui lit son entrée par un ReadFile bloquant,
// n'y voit qu'une fin de flux — le shell se croit alors sans clavier et rend la main
// aussitôt.
func run(rc, keys, cmd string, cols, rows int, settle, pause, last time.Duration) (string, error) {
	entreeL, entreeE, err := tuyau()
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(entreeE)

	sortieL, sortieE, err := tuyau()
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(sortieL)

	var console windows.Handle
	taille := windows.Coord{X: int16(cols), Y: int16(rows)}
	if err := windows.CreatePseudoConsole(taille, entreeL, sortieE, 0, &console); err != nil {
		return "", fmt.Errorf("CreatePseudoConsole : %w", err)
	}
	// Refermer deux fois un pseudo-terminal corrompt le tas : on ne le ferme qu'une, que
	// ce soit ici ou plus bas.
	defer func() {
		if console != 0 {
			windows.ClosePseudoConsole(console)
			console = 0
		}
	}()

	// Le pseudo-terminal démarre son propre conhost, et il lui faut un instant avant de
	// pomper l'entrée. Un shell lancé trop tôt lit une fin de flux là où il attendait un
	// clavier — et rend la main aussitôt.
	time.Sleep(700 * time.Millisecond)

	// Le rc est chargé par la ligne de commande plutôt que tapé : une attente de moins,
	// et le shell est prêt dès son premier prompt.
	if rc != "" {
		cmd += ` -Command ". '` + rc + `'"`
	}

	pi, err := lancer(console, cmd)
	if err != nil {
		return "", err
	}
	defer windows.TerminateProcess(pi.Process, 0)
	defer windows.CloseHandle(pi.Process)
	defer windows.CloseHandle(pi.Thread)

	// Le pseudo-terminal s'est fait ses propres copies des deux bouts qu'on lui a
	// confiés : garder les nôtres empêcherait la lecture de finir quand le shell meurt,
	// et brouille le décompte des écrivains côté entrée.
	windows.CloseHandle(sortieE)
	windows.CloseHandle(entreeL)

	// Lecture continue : le tampon du tuyau est petit, et un cadre fait quelques
	// kilo-octets. Sans lecteur, le shell se bloquerait en écriture.
	ecran := make(chan string, 1)
	go func() {
		var b strings.Builder
		buf := make([]byte, 8192)
		for {
			var n uint32
			if err := windows.ReadFile(sortieL, buf, &n, nil); err != nil || n == 0 {
				break
			}
			b.Write(buf[:n])
		}
		ecran <- b.String()
	}()

	time.Sleep(settle)

	// Touche par touche, comme un humain : c'est la seule façon d'exercer les
	// gestionnaires de PSReadLine un par un.
	for _, touche := range decouper(keys) {
		if err := taper(entreeE, touche); err != nil {
			return "", fmt.Errorf("frappe %q : %w", touche, err)
		}
		time.Sleep(pause)
	}
	time.Sleep(last)

	// Un shell mort explique un écran muet mieux que n'importe quelle hypothèse sur les
	// touches. 259 = STILL_ACTIVE.
	var code uint32
	if err := windows.GetExitCodeProcess(pi.Process, &code); err == nil && code != 259 {
		fmt.Fprintf(os.Stderr, "harnais : le shell a rendu la main (code %d)\n", code)
	}

	windows.TerminateProcess(pi.Process, 0)
	windows.ClosePseudoConsole(console)
	console = 0

	select {
	case s := <-ecran:
		return s, nil
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("le shell n'a pas rendu la main")
	}
}

// tuyau crée un tuyau anonyme synchrone, celui qu'attend ConPTY.
func tuyau() (lecture, ecriture windows.Handle, err error) {
	err = windows.CreatePipe(&lecture, &ecriture, nil, 0)
	return
}

func taper(h windows.Handle, s string) error {
	b := []byte(s)
	for len(b) > 0 {
		var n uint32
		if err := windows.WriteFile(h, b, &n, nil); err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

// lancer démarre le shell attaché au pseudo-terminal. C'est l'attribut
// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE qui fait tout : le processus fils croit alors
// parler à une vraie console, et PSReadLine s'anime.
func lancer(console windows.Handle, cmd string) (*windows.ProcessInformation, error) {
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, err
	}
	defer attrs.Delete()

	// L'attribut attend la *valeur* du pseudo-terminal là où l'API demande un pointeur :
	// c'est la convention de UpdateProcThreadAttribute pour les attributs scalaires. On
	// la dépose donc telle quelle dans la variable, plutôt que par une conversion
	// entier → pointeur, que `go vet` refuse — à juste titre partout ailleurs.
	var valeur unsafe.Pointer
	*(*windows.Handle)(unsafe.Pointer(&valeur)) = console
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		valeur, unsafe.Sizeof(console)); err != nil {
		return nil, fmt.Errorf("PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE : %w", err)
	}

	var si windows.StartupInfoEx
	si.ProcThreadAttributeList = attrs.List()
	si.Cb = uint32(unsafe.Sizeof(si))

	// Nom d'application nul : c'est alors la ligne de commande qui nomme le programme,
	// et CreateProcess le cherche dans le PATH — ce qu'il ne fait pas autrement.
	ligne, err := windows.UTF16PtrFromString(cmd)
	if err != nil {
		return nil, err
	}

	// Sans STARTF_USESTDHANDLES, le fils hérite des flux standard du père. Or les nôtres
	// sont des tuyaux (on est lancé depuis un script), et le fils s'y retrouve avec une
	// entrée standard déjà close : il lit une fin de flux au lieu du clavier du
	// pseudo-terminal, et rend la main aussitôt. On les efface le temps de la création
	// pour que la console prenne le relais, puis on les remet.
	sauve := [3]windows.Handle{}
	flux := [3]uint32{windows.STD_INPUT_HANDLE, windows.STD_OUTPUT_HANDLE, windows.STD_ERROR_HANDLE}
	for i, f := range flux {
		sauve[i], _ = windows.GetStdHandle(f)
		windows.SetStdHandle(f, 0)
	}

	var pi windows.ProcessInformation
	err = windows.CreateProcess(nil, ligne, nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT,
		nil, nil, &si.StartupInfo, &pi)

	for i, f := range flux {
		windows.SetStdHandle(f, sauve[i])
	}
	if err != nil {
		return nil, fmt.Errorf("CreateProcess : %w", err)
	}
	return &pi, nil
}

// decouper découpe une suite de frappes en touches. Une flèche est trois octets
// (ESC [ B) qu'un vrai clavier envoie d'un seul tenant : les espacer ferait voir à
// PSReadLine un Échap suivi de deux lettres — et une fois passé son délai
// d'échappement, c'est bien ainsi qu'il les traiterait.
func decouper(keys string) []string {
	var touches []string
	runes := []rune(keys)
	for i := 0; i < len(runes); i++ {
		if runes[i] != 0x1b {
			touches = append(touches, string(runes[i]))
			continue
		}
		fin := i + 1
		if fin < len(runes) && (runes[fin] == '[' || runes[fin] == 'O') {
			fin++
			for fin < len(runes) && (runes[fin] < '@' || runes[fin] > '~') {
				fin++
			}
			if fin < len(runes) {
				fin++
			}
		} else if fin < len(runes) {
			fin++
		}
		touches = append(touches, string(runes[i:fin]))
		i = fin - 1
	}
	return touches
}

// unescape lit les échappements Go d'une chaîne de touches (\t, \r, \x1b…).
func unescape(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	return strconv.Unquote(`"` + strings.ReplaceAll(s, `"`, `\"`) + `"`)
}

// ansi couvre ce que PSReadLine et jigger émettent : séquences CSI, OSC, et les
// sauvegardes/restaurations de curseur.
var ansi = regexp.MustCompile("\x1b\\][^\x07\x1b]*(\x07|\x1b\\\\)|\x1b\\[[0-9;?]*[ -/]*[@-~]|\x1b[78=>]|\x1b\\([AB0]")

// strip rend ce que l'œil verrait : le texte, sans les séquences d'échappement.
func strip(s string) string { return ansi.ReplaceAllString(s, "") }
