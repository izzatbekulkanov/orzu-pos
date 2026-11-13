package main

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/izzatbekulkanov/orzu-pos/internal/tray"
)

func main() {
	targetURL := "http://localhost:8080"

	// 1. Agar server yoqilmagan bo'lsa, uni fonda (terminal oynamasisiz) ishga tushirish
	if !isServerUp(targetURL) {
		startServerSilently()
		// Server yuklanishini 3 soniyagacha kutish
		for i := 0; i < 6; i++ {
			time.Sleep(500 * time.Millisecond)
			if isServerUp(targetURL) {
				break
			}
		}
	}

	// 2. Kassa oynasini terminalsiz, toza ilova sifatida ochish
	tray.OpenApp(targetURL)
}

func isServerUp(url string) bool {
	client := http.Client{Timeout: 600 * time.Millisecond}
	resp, err := client.Get(url + "/api/settings")
	if err == nil && resp.StatusCode == 200 {
		resp.Body.Close()
		return true
	}
	return false
}

func startServerSilently() {
	exeDir, _ := os.Executable()
	baseDir := filepath.Dir(exeDir)

	candidates := []string{
		filepath.Join(baseDir, "orzu-server.exe"),
		filepath.Join(`D:\ORZU_POS`, "orzu-server.exe"),
		filepath.Join("bin", "orzu-server.exe"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			cmd := exec.Command(path)
			cmd.Dir = filepath.Dir(path)
			cmd.SysProcAttr = &syscall.SysProcAttr{
				HideWindow:    true,
				CreationFlags: 0x08000000, // CREATE_NO_WINDOW
			}
			_ = cmd.Start()
			return
		}
	}
}
