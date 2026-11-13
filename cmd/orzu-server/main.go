package main

import (
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/izzatbekulkanov/orzu-pos/internal/api"
	"github.com/izzatbekulkanov/orzu-pos/internal/database"
	"github.com/izzatbekulkanov/orzu-pos/internal/tray"
	"github.com/izzatbekulkanov/orzu-pos/web"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("   🛒 ORZU POS — Olmos Orzu market Server v1.0   ")
	fmt.Println("   Namangan davlat pedagogika instituti RTTM      ")
	fmt.Println("==================================================")

	// 1. Ma'lumotlar bazasini ishga tushirish (PostgreSQL yoki SQLite)
	_, err := database.InitDB()
	if err != nil {
		log.Fatalf("❌ Ma'lumotlar bazasida jiddiy xato: %v\n", err)
	}

	app := fiber.New(fiber.Config{
		AppName:      "ORZU POS - Olmos Orzu market",
		ServerHeader: "OrzuServer/1.0",
	})

	// 2. API marshrutlarini sozlash
	api.SetupRoutes(app)

	// 3. Embedded veb interfeyslarni ulash
	kassaFS, _ := fs.Sub(web.WebFS, "kassa")
	adminFS, _ := fs.Sub(web.WebFS, "admin")

	// / -> Kassa ilovasi
	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(kassaFS),
		Index:        "index.html",
		NotFoundFile: "index.html",
		Browse:       false,
	}))

	// /admin -> Boshqaruv paneli
	app.Use("/admin", filesystem.New(filesystem.Config{
		Root:         http.FS(adminFS),
		Index:        "index.html",
		NotFoundFile: "index.html",
		Browse:       false,
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Lokal tarmoq (LAN) IP manzillarini aniqlash
	printServerAddresses(port)

	log.Printf("🚀 ORZU Server ishga tushdi: port :%s\n", port)
	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Serverni ishga tushirishda xato: %v\n", err)
		}
	}()

	// Windows tizimida pastki o'ng burchakdagi (System Tray) belgini ishga tushirish
	if err := tray.InitTray(port); err != nil {
		select {}
	}
}

func printServerAddresses(port string) {
	fmt.Printf("\n✨ Kassa ilovasi manzili:      http://localhost:%s\n", port)
	fmt.Printf("📊 Do'kon boshqaruv paneli:     http://localhost:%s/admin\n\n", port)

	addrs, err := net.InterfaceAddrs()
	if err == nil {
		fmt.Println("🌐 Lokal tarmoqdagi boshqa kassa va telefonlar uchun ulanish manzillari:")
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					fmt.Printf("   👉 http://%s:%s\n", ipNet.IP.String(), port)
				}
			}
		}
	}
	fmt.Println("--------------------------------------------------\n")
}
