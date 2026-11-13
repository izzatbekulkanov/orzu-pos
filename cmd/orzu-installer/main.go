package main

import (
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
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/izzatbekulkanov/orzu-pos/web"
)

var defaultInstallDir = `D:\ORZU_POS`

func main() {
	fmt.Println("==================================================")
	fmt.Println("   📦 ORZU POS — O'rnatuvchi Markazi (Setup) v1.0 ")
	fmt.Println("   Olmos Orzu market modullar menejeri            ")
	fmt.Println("==================================================")

	if envDir := os.Getenv("ORZU_INSTALL_DIR"); envDir != "" {
		defaultInstallDir = envDir
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Setup API routes
	setupInstallerAPI(app)

	// Web UI
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
		fmt.Printf("🚀 O'rnatuvchi oynasi ochilmoqda: %s\n", url)
		openAppWindow(url)
	}()

	fmt.Printf("O'rnatuvchi xizmati faol: %s\n", url)
	if err := app.Listen("127.0.0.1:" + port); err != nil {
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
					"installed": isFileExists(filepath.Join(defaultInstallDir, "orzu-db.exe")) || isFileExists("bin/orzu-db.exe"),
					"running":   false,
				},
				"server": fiber.Map{
					"installed": isFileExists(filepath.Join(defaultInstallDir, "orzu-server.exe")) || isFileExists("bin/orzu-server.exe"),
					"running":   isServerRunning(),
				},
				"kassa": fiber.Map{
					"installed": isFileExists(filepath.Join(defaultInstallDir, "orzu-kassa.exe")) || isFileExists("bin/orzu-kassa.exe"),
					"running":   false,
				},
				"backup": fiber.Map{
					"installed": isFileExists(filepath.Join(defaultInstallDir, "orzu-backup.exe")) || isFileExists("bin/orzu-backup.exe"),
					"running":   false,
				},
			},
		}
		return c.JSON(status)
	})

	api.Post("/install/:mod", func(c *fiber.Ctx) error {
		mod := c.Params("mod")
		if err := copyModule(mod); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		_ = createDesktopShortcut(mod)
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

		// DB Init
		go func() {
			cmd := exec.Command(filepath.Join(defaultInstallDir, "orzu-db.exe"))
			cmd.Dir = defaultInstallDir
			_ = cmd.Run()
		}()

		return c.JSON(fiber.Map{"success": true, "message": "Barcha modullar o'rnatildi!"})
	})

	api.Post("/run/:mod", func(c *fiber.Ctx) error {
		mod := c.Params("mod")
		exeName := fmt.Sprintf("orzu-%s.exe", mod)
		targetPath := filepath.Join(defaultInstallDir, exeName)
		if !isFileExists(targetPath) {
			targetPath = filepath.Join("bin", exeName)
		}
		if !isFileExists(targetPath) {
			return c.Status(404).JSON(fiber.Map{"error": "Fayl topilmadi"})
		}

		cmd := exec.Command(targetPath)
		cmd.Dir = filepath.Dir(targetPath)
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
	exeName := fmt.Sprintf("orzu-%s.exe", mod)

	srcPath := filepath.Join("bin", exeName)
	destPath := filepath.Join(defaultInstallDir, exeName)

	if !isFileExists(srcPath) {
		return fmt.Errorf("%s manba fayli topilmadi", srcPath)
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
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

	var name, target string
	switch mod {
	case "kassa":
		name = "ORZU Kassa"
		target = filepath.Join(defaultInstallDir, "orzu-kassa.exe")
	case "admin":
		name = "ORZU Boshqaruv"
		target = "http://localhost:8080/admin"
	case "server":
		name = "ORZU Server"
		target = filepath.Join(defaultInstallDir, "orzu-server.exe")
	default:
		return nil
	}

	if strings.HasPrefix(target, "http") {
		// Create .url internet shortcut
		urlContent := fmt.Sprintf("[InternetShortcut]\nURL=%s\nIconIndex=0\n", target)
		return os.WriteFile(filepath.Join(desktopDir, name+".url"), []byte(urlContent), 0644)
	}

	// Create Windows .lnk via PowerShell WScript.Shell
	psScript := fmt.Sprintf(`$wsh = New-Object -ComObject WScript.Shell; $s = $wsh.CreateShortcut("%s"); $s.TargetPath = "%s"; $s.WorkingDirectory = "%s"; $s.Save()`,
		filepath.Join(desktopDir, name+".lnk"), target, defaultInstallDir)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	return cmd.Run()
}

func openAppWindow(url string) {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "start", "msedge", "--app="+url)
		_ = cmd.Start()
	} else {
		cmd := exec.Command("xdg-open", url)
		_ = cmd.Start()
	}
}
