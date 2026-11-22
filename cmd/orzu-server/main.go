package main

import (
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/izzatbekulkanov/orzu-pos/internal/api"
	"github.com/izzatbekulkanov/orzu-pos/internal/database"
	"github.com/izzatbekulkanov/orzu-pos/internal/tray"
	"github.com/izzatbekulkanov/orzu-pos/web"
)

func main() {
	fmt.Println("\n=======================================================")
	fmt.Println("   ORZU POS - Olmos Orzu market Server v1.0            ")
	fmt.Println("   Jonli Terminal & So'rovlar Logi                     ")
	fmt.Println("   Namangan davlat pedagogika instituti RTTM           ")
	fmt.Println("=======================================================")

	// 1. Ma'lumotlar bazasini ishga tushirish (PostgreSQL yoki SQLite)
	_, err := database.InitDB()
	if err != nil {
		log.Fatalf("[Xatolik] Ma'lumotlar bazasida jiddiy xato: %v\n", err)
	}

	app := fiber.New(fiber.Config{
		AppName:      "ORZU POS - Olmos Orzu market",
		ServerHeader: "OrzuServer/1.0",
	})

	// 2. API marshrutlarini sozlash
	api.SetupRoutes(app)

	// 3. Keshni oqilona boshqarish: HTML har doim yangi bo'lsin, og'ir JS/CSS kutubxonalari esa xotirada keshlanib tez yuklansin
	app.Use(func(c *fiber.Ctx) error {
		path := c.Path()
		if strings.HasSuffix(path, ".html") || path == "/" || path == "/admin" || path == "/db" || path == "/backup" {
			c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Set("Pragma", "no-cache")
			c.Set("Expires", "0")
		}
		return c.Next()
	})

	// Mahalliy diskdagi web fayllarni ulash (Jonli yangilanishlar uchun)
	localWebDirs := []string{`D:\ORZU_POS\web`, `web`}
	for _, ldir := range localWebDirs {
		if fi, err := os.Stat(ldir); err == nil && fi.IsDir() {
			app.Static("/shared", filepath.Join(ldir, "shared"), fiber.Static{
				Compress:  false,
				ByteRange: true,
				MaxAge:    0,
			})
			break
		}
	}

	// Embedded veb interfeyslarni ulash (Zaxira sifatida)
	kassaFS, _ := fs.Sub(web.WebFS, "kassa")
	adminFS, _ := fs.Sub(web.WebFS, "admin")
	iconsFS, _ := fs.Sub(web.WebFS, "icons")
	sharedFS, _ := fs.Sub(web.WebFS, "shared")
	dbFS, _ := fs.Sub(web.WebFS, "db")
	backupFS, _ := fs.Sub(web.WebFS, "backup")

	// /shared -> 100% Oflayn Resurslar (Tailwind, Lucide, CSS)
	app.Use("/shared", filesystem.New(filesystem.Config{
		Root:   http.FS(sharedFS),
		Browse: false,
	}))

	// /uploads -> Oflayn mahalliy tovar rasmlari
	uploadsDir := `D:\ORZU_POS\uploads`
	_ = os.MkdirAll(filepath.Join(uploadsDir, "products"), 0755)
	_ = os.MkdirAll(filepath.Join("uploads", "products"), 0755)
	app.Static("/uploads", uploadsDir)
	app.Static("/uploads", "./uploads")

	// /manifest.json -> PWA Manifest
	app.Get("/manifest.json", func(c *fiber.Ctx) error {
		manifestBytes, err := web.WebFS.ReadFile("manifest.json")
		if err != nil {
			return c.SendStatus(404)
		}
		c.Set("Content-Type", "application/json")
		return c.Send(manifestBytes)
	})

	// /icons -> Ikonkalar
	app.Use("/icons", filesystem.New(filesystem.Config{
		Root:   http.FS(iconsFS),
		Browse: false,
	}))

	// /admin -> Boshqaruv paneli
	app.Use("/admin", filesystem.New(filesystem.Config{
		Root:         http.FS(adminFS),
		Index:        "index.html",
		NotFoundFile: "index.html",
		Browse:       false,
	}))

	// /db -> Ma'lumotlar bazasi boshqaruvi
	app.Use("/db", filesystem.New(filesystem.Config{
		Root:         http.FS(dbFS),
		Index:        "index.html",
		NotFoundFile: "index.html",
		Browse:       false,
	}))

	// /backup -> Arxiv va zaxira
	app.Use("/backup", filesystem.New(filesystem.Config{
		Root:         http.FS(backupFS),
		Index:        "index.html",
		NotFoundFile: "index.html",
		Browse:       false,
	}))

	// / -> Kassa ilovasi
	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(kassaFS),
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

	log.Printf("[Server] ORZU Server ishga tushdi: port :%s\n", port)
	
	// Windows tizimida pastki o'ng burchakdagi (System Tray) belgini ishga tushirish
	go func() {
		_ = tray.InitTray(port)
	}()

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Serverni ishga tushirishda xato: %v\n", err)
	}
}

func printServerAddresses(port string) {
	fmt.Printf("\nKassa ilovasi manzili:      http://localhost:%s\n", port)
	fmt.Printf("Do'kon boshqaruv paneli:    http://localhost:%s/admin\n\n", port)

	addrs, err := net.InterfaceAddrs()
	if err == nil {
		fmt.Println("Lokal tarmoqdagi boshqa kassa va telefonlar uchun ulanish manzillari:")
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					fmt.Printf("   http://%s:%s\n", ipNet.IP.String(), port)
				}
			}
		}
	}
	fmt.Println("--------------------------------------------------\n")
}
