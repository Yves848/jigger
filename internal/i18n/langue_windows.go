//go:build windows

package i18n

import (
	"syscall"
	"unsafe"
)

// cultureSysteme rend la langue de l'interface utilisateur Windows (« fr-FR »), là où les
// variables POSIX n'existent pas. Appel direct à l'API, sans dépendance ajoutée — comme
// le fait déjà internal/pm/proc_windows.go.
func cultureSysteme() string {
	const tailleMax = 85 // LOCALE_NAME_MAX_LENGTH
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultLocaleName")

	buf := make([]uint16, tailleMax)
	n, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(tailleMax))
	if n <= 1 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n-1]) // n compte le zéro final
}
