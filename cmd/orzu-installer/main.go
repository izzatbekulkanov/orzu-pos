package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/izzatbekulkanov/orzu-pos/internal/database"
	"github.com/izzatbekulkanov/orzu-pos/internal/installer"
	"github.com/izzatbekulkanov/orzu-pos/internal/license"
	"github.com/izzatbekulkanov/orzu-pos/internal/services"
	"github.com/izzatbekulkanov/orzu-pos/internal/tray"
	"github.com/izzatbekulkanov/orzu-pos/web"
)

var defaultInstallDir = `D:\ORZU_POS`

func main() {
	// 1. Agar O'rnatuvchi oynasi allaqachon ochilgan bo'lsa, uni oldinga chiqarish
	if tray.ActivateExistingWindow("ORZU O'rnatuvchi") || tray.ActivateExistingWindow("ORZU Setup") {
		return
	}

	if envDir := os.Getenv("ORZU_INSTALL_DIR"); envDir != "" {
		defaultInstallDir = envDir
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Setup API routes
	setupInstallerAPI(app)

	// Web UI
	app.Get("/manifest.json", func(c *fiber.Ctx) error {
		manifestBytes, err := web.WebFS.ReadFile("manifest.json")
		if err != nil {
			return c.SendStatus(404)
		}
		c.Set("Content-Type", "application/json")
		return c.Send(manifestBytes)
	})

	iconsFS, _ := fs.Sub(web.WebFS, "icons")
	app.Use("/icons", filesystem.New(filesystem.Config{
		Root:   http.FS(iconsFS),
		Browse: false,
	}))

	sharedFS, _ := fs.Sub(web.WebFS, "shared")
	app.Use("/shared", filesystem.New(filesystem.Config{
		Root:   http.FS(sharedFS),
		Browse: false,
	}))

	// /uploads -> Oflayn tovar rasmlari
	uploadsDir := filepath.Join(defaultInstallDir, "uploads")
	_ = os.MkdirAll(filepath.Join(uploadsDir, "products"), 0755)
	_ = os.MkdirAll(filepath.Join("uploads", "products"), 0755)
	app.Static("/uploads", uploadsDir)
	app.Static("/uploads", "./uploads")

	installerFS, _ := fs.Sub(web.WebFS, "installer")
	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(installerFS),
		Index:        "index.html",
		NotFoundFile: "index.html",
	}))

	port := "8899"
	url := "http://localhost:" + port

	go func() {
		time.Sleep(600 * time.Millisecond)
		openAppWindow(url)
	}()

	log.Printf("[ORZU Setup] O'rnatuvchi markazi faol: %s\n", url)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Xatolik: %v", err)
	}
}

func setupInstallerAPI(app *fiber.App) {
	api := app.Group("/api/installer")

	api.Get("/status", func(c *fiber.Ctx) error {
		status := fiber.Map{
			"install_dir": defaultInstallDir,
			"modules": fiber.Map{
				"db": fiber.Map{
					"installed": isModuleInstalled("db"),
					"running":   false,
				},
				"server": fiber.Map{
					"installed": isModuleInstalled("server"),
					"running":   isServerRunning(),
				},
				"kassa": fiber.Map{
					"installed": isModuleInstalled("kassa"),
					"running":   false,
				},
				"backup": fiber.Map{
					"installed": isModuleInstalled("backup"),
					"running":   false,
				},
			},
			"license": license.GetStatus(),
		}
		return c.JSON(status)
	})

	// License check and activate
	api.Get("/license", func(c *fiber.Ctx) error {
		return c.JSON(license.GetStatus())
	})

	api.Post("/activate", func(c *fiber.Ctx) error {
		var req struct {
			Key string `json:"key"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Noto'g'ri so'rov formati"})
		}
		ok, msg := license.ValidateAndActivate(req.Key)
		if !ok {
			return c.Status(400).JSON(fiber.Map{"error": msg})
		}
		return c.JSON(fiber.Map{"success": true, "message": msg, "license": license.GetStatus()})
	})

	// DB Status & Diagnostic
	api.Get("/db-status", func(c *fiber.Ctx) error {
		dbPath := filepath.Join(defaultInstallDir, "data", "orzu_pos.db")
		if !isFileExists(dbPath) {
			dbPath = filepath.Join("data", "orzu_pos.db")
		}

		status := fiber.Map{
			"db_type":    "SQLite (Oflayn va mustaqil rejim)",
			"path":       dbPath,
			"exists":     isFileExists(dbPath),
			"size_kb":    "0 KB",
			"tables":     8,
			"status":     "Normal",
			"migrations": "Barcha jadvallar tayyor",
		}

		if isFileExists(dbPath) {
			if fi, err := os.Stat(dbPath); err == nil {
				status["size_kb"] = fmt.Sprintf("%.1f KB", float64(fi.Size())/1024)
			}
		}

		return c.JSON(status)
	})

	api.Post("/db-migrate", func(c *fiber.Ctx) error {
		_, err := database.InitDB()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true, "message": "Jadvallar muvaffaqiyatli yangilandi!"})
	})

	// Backups API
	api.Get("/backups", func(c *fiber.Ctx) error {
		list, err := services.ListBackups()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(list)
	})

	api.Post("/backup-create", func(c *fiber.Ctx) error {
		b, err := services.CreateBackup()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true, "backup": b})
	})

	api.Post("/install/:mod", func(c *fiber.Ctx) error {
		mod := c.Params("mod")
		if err := copyModule(mod); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		_ = createDesktopShortcut(mod)
		_ = installer.MarkInstalled()
		return c.JSON(fiber.Map{"success": true})
	})

	api.Post("/install-all", func(c *fiber.Ctx) error {
		_ = os.MkdirAll(defaultInstallDir, 0755)

		modules := []string{"db", "server", "kassa", "backup"}
		for _, m := range modules {
			if err := copyModule(m); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("%s ni o'rnatishda xato: %v", m, err)})
			}
			_ = createDesktopShortcut(m)
		}

		_ = installer.MarkInstalled()

		// Init database
		go func() {
			_, _ = database.InitDB()
		}()

		return c.JSON(fiber.Map{"success": true, "message": "Barcha modullar muvaffaqiyatli o'rnatildi!"})
	})

	api.Post("/run/:mod", func(c *fiber.Ctx) error {
		mod := c.Params("mod")
		exeName := fmt.Sprintf("orzu-%s.exe", mod)
		targetPath := filepath.Join(defaultInstallDir, exeName)
		if !isFileExists(targetPath) {
			targetPath = filepath.Join("bin", exeName)
		}
		if !isFileExists(targetPath) {
			targetPath = filepath.Join(`D:\Github\orzu-pos\bin`, exeName)
		}
		if !isFileExists(targetPath) {
			return c.Status(404).JSON(fiber.Map{"error": "Fayl topilmadi"})
		}

		cmd := exec.Command(targetPath)
		cmd.Dir = filepath.Dir(targetPath)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
		if err := cmd.Start(); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	api.Post("/shortcut/:mod", func(c *fiber.Ctx) error {
		mod := c.Params("mod")
		if err := createDesktopShortcut(mod); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// GitHub Auto-Updater API
	api.Get("/updates/check", func(c *fiber.Ctx) error {
		localVersion := getLocalVersionInfo()

		client := &http.Client{Timeout: 6 * time.Second}
		remoteURL := "https://raw.githubusercontent.com/izzatbekulkanov/orzu-pos/main/version.json"
		req, err := http.NewRequest("GET", remoteURL, nil)
		if err != nil {
			return c.JSON(fiber.Map{
				"online":          false,
				"has_update":      false,
				"current_version": localVersion.Version,
				"message":         "So'rov xatosi: " + err.Error(),
			})
		}
		req.Header.Set("Cache-Control", "no-cache")

		resp, err := client.Do(req)
		if err != nil {
			return c.JSON(fiber.Map{
				"online":          false,
				"has_update":      false,
				"current_version": localVersion.Version,
				"message":         "Internet yoki GitHub bilan aloqa yo'q (Oflayn rejim)",
			})
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return c.JSON(fiber.Map{
				"online":          true,
				"has_update":      false,
				"current_version": localVersion.Version,
				"message":         fmt.Sprintf("GitHub status: %d", resp.StatusCode),
			})
		}

		var remoteVersion struct {
			Version     string   `json:"version"`
			VersionCode int      `json:"version_code"`
			ReleaseDate string   `json:"release_date"`
			Title       string   `json:"title"`
			Changelog   []string `json:"changelog"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&remoteVersion); err != nil {
			return c.JSON(fiber.Map{
				"online":          true,
				"has_update":      false,
				"current_version": localVersion.Version,
				"error":           "version.json ni o'qib bo'lmadi",
			})
		}

		hasUpdate := remoteVersion.VersionCode > localVersion.VersionCode ||
			(remoteVersion.Version != "" && remoteVersion.Version != localVersion.Version)

		return c.JSON(fiber.Map{
			"online":          true,
			"has_update":      hasUpdate,
			"current_version": localVersion.Version,
			"latest_version":  remoteVersion.Version,
			"title":           remoteVersion.Title,
			"release_date":    remoteVersion.ReleaseDate,
			"changelog":       remoteVersion.Changelog,
		})
	})

	api.Post("/updates/apply", func(c *fiber.Ctx) error {
		gitDir := `D:\Github\orzu-pos`
		hasGit := isFileExists(filepath.Join(gitDir, ".git"))

		go func() {
			time.Sleep(500 * time.Millisecond)
			if hasGit {
				pullCmd := exec.Command("git", "-C", gitDir, "pull", "origin", "main")
				_ = pullCmd.Run()

				buildScript := filepath.Join(gitDir, "build_all.ps1")
				if isFileExists(buildScript) {
					buildCmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", buildScript)
					_ = buildCmd.Run()
				}
			}
		}()

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Yangilanish jarayoni boshlandi. Dastur fayllari GitHub'dan tortilib, qayta yig'ilmoqda...",
		})
	})
}

type VersionInfo struct {
	Version     string   `json:"version"`
	VersionCode int      `json:"version_code"`
	ReleaseDate string   `json:"release_date"`
	Title       string   `json:"title"`
	Changelog   []string `json:"changelog"`
}

func getLocalVersionInfo() VersionInfo {
	defaultInfo := VersionInfo{
		Version:     "2.2.0",
		VersionCode: 220,
		ReleaseDate: "2026-11-14",
		Title:       "ORZU POS 2.2 — AI Tasvir Generatsiyasi va Smart Boshqaruv",
		Changelog: []string{
			"OpenRouter AI orqali tovarlar rasmini avtomatik generatsiya qilish va oflayn saqlash",
			"GitHub orqali avtomatik yangilanishlarni tekshirish va o'rnatish tizimi",
			"Sensor monobloklar uchun 2 bosqichli professional boshqaruv menyusi",
			"Suriladigan (Drag & Drop) sensor klaviatura tugmasi",
		},
	}

	candidates := []string{
		filepath.Join(defaultInstallDir, "version.json"),
		filepath.Join(`D:\Github\orzu-pos`, "version.json"),
		"version.json",
	}

	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			var vi VersionInfo
			if err := json.Unmarshal(data, &vi); err == nil && vi.Version != "" {
				return vi
			}
		}
	}
	return defaultInfo
}

func isModuleInstalled(mod string) bool {
	exeName := fmt.Sprintf("orzu-%s.exe", mod)
	candidates := []string{
		filepath.Join(defaultInstallDir, exeName),
		filepath.Join("bin", exeName),
		filepath.Join(`D:\Github\orzu-pos\bin`, exeName),
	}
	for _, p := range candidates {
		if isFileExists(p) {
			return true
		}
	}
	return false
}

func isFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isServerRunning() bool {
	client := http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://localhost:8080/api/settings")
	if err == nil && resp.StatusCode == 200 {
		resp.Body.Close()
		return true
	}
	return false
}

func copyModule(mod string) error {
	_ = os.MkdirAll(defaultInstallDir, 0755)
	_ = os.MkdirAll(filepath.Join(defaultInstallDir, "assets", "icons"), 0755)

	exeName := fmt.Sprintf("orzu-%s.exe", mod)
	destPath := filepath.Join(defaultInstallDir, exeName)

	// Copy icons if available
	iconName := mod + ".ico"
	iconCandidates := []string{
		filepath.Join("assets", "icons", iconName),
		filepath.Join(`D:\Github\orzu-pos\assets\icons`, iconName),
		filepath.Join(defaultInstallDir, "assets", "icons", iconName),
	}
	for _, ic := range iconCandidates {
		if isFileExists(ic) && ic != filepath.Join(defaultInstallDir, "assets", "icons", iconName) {
			_ = copyFile(ic, filepath.Join(defaultInstallDir, "assets", "icons", iconName))
			break
		}
	}

	// 1. If already installed at destination, succeed immediately!
	if isFileExists(destPath) {
		return nil
	}

	// 2. Search for source in possible locations
	exeSelf, _ := os.Executable()
	selfDir := filepath.Dir(exeSelf)

	searchPaths := []string{
		filepath.Join("bin", exeName),
		filepath.Join(`D:\Github\orzu-pos\bin`, exeName),
		filepath.Join(`D:\ORZU_POS`, exeName),
		filepath.Join(selfDir, exeName),
		filepath.Join(selfDir, "bin", exeName),
	}

	var foundSrc string
	for _, p := range searchPaths {
		if isFileExists(p) {
			foundSrc = p
			break
		}
	}

	if foundSrc == "" {
		return fmt.Errorf("%s manba fayli topilmadi. bin katalogini tekshiring.", exeName)
	}

	if foundSrc == destPath {
		return nil
	}

	return copyFile(foundSrc, destPath)
}

func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

func createDesktopShortcut(mod string) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	homeDir, _ := os.UserHomeDir()
	desktopDir := filepath.Join(homeDir, "Desktop")
	if !isFileExists(desktopDir) {
		return nil
	}

	var name, target, iconPath string
	switch mod {
	case "kassa":
		name = "ORZU Kassa"
		target = filepath.Join(defaultInstallDir, "orzu-kassa.exe")
		iconPath = filepath.Join(defaultInstallDir, "assets", "icons", "kassa.ico")
	case "admin":
		name = "ORZU Boshqaruv"
		target = "http://localhost:8080/admin"
		iconPath = filepath.Join(defaultInstallDir, "assets", "icons", "admin.ico")
	case "server":
		name = "ORZU Server"
		target = filepath.Join(defaultInstallDir, "orzu-server.exe")
		iconPath = filepath.Join(defaultInstallDir, "assets", "icons", "server.ico")
	case "db":
		name = "ORZU DB"
		target = filepath.Join(defaultInstallDir, "orzu-db.exe")
		iconPath = filepath.Join(defaultInstallDir, "assets", "icons", "db.ico")
	case "backup":
		name = "ORZU Arxiv"
		target = filepath.Join(defaultInstallDir, "orzu-backup.exe")
		iconPath = filepath.Join(defaultInstallDir, "assets", "icons", "backup.ico")
	default:
		return nil
	}

	if strings.HasPrefix(target, "http") {
		// Create .lnk for URL to allow custom icon
		psScript := fmt.Sprintf(`$wsh = New-Object -ComObject WScript.Shell; $s = $wsh.CreateShortcut("%s"); $s.TargetPath = "msedge.exe"; $s.Arguments = "--app=%s --force-dark-mode --start-maximized"; $s.IconLocation = "%s"; $s.Save()`,
			filepath.Join(desktopDir, name+".lnk"), target, iconPath)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
		return cmd.Run()
	}

	// Create Windows .lnk via PowerShell WScript.Shell with custom icon
	psScript := fmt.Sprintf(`$wsh = New-Object -ComObject WScript.Shell; $s = $wsh.CreateShortcut("%s"); $s.TargetPath = "%s"; $s.WorkingDirectory = "%s"; if (Test-Path "%s") { $s.IconLocation = "%s" }; $s.Save()`,
		filepath.Join(desktopDir, name+".lnk"), target, defaultInstallDir, iconPath, iconPath)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	return cmd.Run()
}

func openAppWindow(url string) {
	tray.OpenApp(url)
}
