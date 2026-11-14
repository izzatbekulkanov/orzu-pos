package installer

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32DLL          = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW    = user32DLL.NewProc("MessageBoxW")
	defaultInstallDir  = `D:\ORZU_POS`
	installedFlagPath  = filepath.Join(defaultInstallDir, "config", "installed.flag")
)

// IsSystemInstalled checks if ORZU POS has been installed via the installer
func IsSystemInstalled() bool {
	// 1. Check explicit installed.flag
	if _, err := os.Stat(installedFlagPath); err == nil {
		return true
	}
	// 2. Check if database file exists and is initialized
	dbPath := filepath.Join(defaultInstallDir, "data", "orzu_pos.db")
	if fi, err := os.Stat(dbPath); err == nil && fi.Size() > 4096 {
		// Mark flag for future fast checks
		_ = MarkInstalled()
		return true
	}
	// Fallback to local dev workspace
	if _, err := os.Stat(filepath.Join("data", "orzu_pos.db")); err == nil {
		return true
	}
	return false
}

// MarkInstalled marks system as installed
func MarkInstalled() error {
	_ = os.MkdirAll(filepath.Dir(installedFlagPath), 0755)
	return os.WriteFile(installedFlagPath, []byte("installed=true\ntimestamp="+time.Now().Format(time.RFC3339)+"\n"), 0644)
}

// EnsureInstalled checks if installed; if not, alerts the user and opens the installer
func EnsureInstalled(appName string) bool {
	if IsSystemInstalled() {
		return true
	}

	// Show Windows Alert Box
	msg := fmt.Sprintf("Iltimos, avval o'rnatuvchini o'rnating!\n\nORZU POS tizimi hali o'rnatilmagan yoki sozlanmagan.\n%s ilovasi ishlashi uchun avval ORZU O'rnatuvchi (Setup) markazini ishga tushiring.", appName)
	title := "ORZU POS - O'rnatish talab etiladi"

	msgPtr, _ := syscall.UTF16PtrFromString(msg)
	titlePtr, _ := syscall.UTF16PtrFromString(title)

	// MB_OK (0x00) | MB_ICONWARNING (0x30) | MB_TOPMOST (0x40000)
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(msgPtr)), uintptr(unsafe.Pointer(titlePtr)), uintptr(0x30|0x40000))

	// Launch Installer
	launchInstaller()
	return false
}

func launchInstaller() {
	// Check if installer is already serving on 8899
	client := http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://localhost:8899/api/installer/status")
	if err == nil && resp.StatusCode == 200 {
		resp.Body.Close()
		openURL("http://localhost:8899")
		return
	}

	candidates := []string{
		filepath.Join(defaultInstallDir, "ORZU_Setup.exe"),
		filepath.Join("bin", "ORZU_Setup.exe"),
		filepath.Join(`D:\Github\orzu-pos\bin`, "ORZU_Setup.exe"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			cmd := exec.Command(p)
			cmd.Dir = filepath.Dir(p)
			cmd.SysProcAttr = &syscall.SysProcAttr{
				HideWindow:    true,
				CreationFlags: 0x08000000,
			}
			_ = cmd.Start()
			time.Sleep(800 * time.Millisecond)
			openURL("http://localhost:8899")
			return
		}
	}

	openURL("http://localhost:8899")
}

func openURL(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
}
