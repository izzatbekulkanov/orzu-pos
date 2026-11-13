package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("   🛒 ORZU Kassa — Sotuv Ilovasi (Kiosk) v1.0    ")
	fmt.Println("   Olmos Orzu market - Kassa moduli               ")
	fmt.Println("==================================================")

	targetURL := "http://localhost:8080"

	// Server faolligini tekshirish
	fmt.Println("⏳ ORZU Server aloqasi tekshirilmoqda...")
	isUp := false
	for i := 0; i < 5; i++ {
		resp, err := http.Get(targetURL + "/api/settings")
		if err == nil && resp.StatusCode == 200 {
			isUp = true
			resp.Body.Close()
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !isUp {
		log.Println("ℹ️ ORZU Server hali yoqilmagan ko'rinadi. Iltimos, avval 'orzu-server.exe' ni ishga tushiring.")
	}

	fmt.Printf("🚀 Kassa ekrani ochilmoqda: %s\n", targetURL)
	openKiosk(targetURL)
}

func openKiosk(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Microsoft Edge kiosk yoki Chrome kiosk rejimida ochish
		cmd = exec.Command("cmd", "/c", "start", "msedge", "--app="+url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // Linux
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
