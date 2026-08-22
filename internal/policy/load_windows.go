//go:build windows

package policy

import (
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetLongPathNameW = kernel32.NewProc("GetLongPathNameW")
)

// resolveRealPath expands 8.3 short names and symlinked path components on
// Windows so containment checks compare canonical forms.
func resolveRealPath(p string) (string, bool) {
	if p == "" {
		return "", false
	}
	ptr, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		return "", false
	}
	buf := make([]uint16, 32768)
	n, _, _ := procGetLongPathNameW.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if n == 0 || n > uintptr(len(buf)) {
		return "", false
	}
	return syscall.UTF16ToString(buf[:n]), true
}
