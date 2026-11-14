package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/izzatbekulkanov/orzu-pos/internal/installer"
	"github.com/izzatbekulkanov/orzu-pos/internal/tray"
)

func main() {
	// 1. Agar Arxiv oynasi allaqachon ochilgan bo'lsa, uni oldinga chiqarish
	if tray.ActivateExistingWindow("ORZU Arxiv") {
		return
	}

	// 2. O'rnatuvchi o'rnatilganligini tekshirish
	if !installer.EnsureInstalled("ORZU Arxiv") {
		return
	}

	// 3. Tezkor avtomatik zaxira nusxa yaratish
	createQuickBackup()

	baseURL := "http://localhost:8080"
	targetURL := "http://localhost:8080/backup"

	// 4. Serverni tekshirish
	if !isServerUp(baseURL) {
		startServerSilently()
		for i := 0; i < 6; i++ {
			time.Sleep(500 * time.Millisecond)
			if isServerUp(baseURL) {
				break
			}
		}
	}

	// 5. Arxiv darchasini ochish
	tray.OpenApp(targetURL)
}

func createQuickBackup() {
	backupDir := `D:\ORZU_POS\backups`
	_ = os.MkdirAll(backupDir, 0755)

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	archiveName := filepath.Join(backupDir, fmt.Sprintf("orzu_pos_backup_%s.zip", timestamp))

	dbFile := `D:\ORZU_POS\data\orzu_pos.db`
	if _, err := os.Stat(dbFile); err != nil {
		dbFile = filepath.Join("data", "orzu_pos.db")
	}

	if fi, err := os.Stat(dbFile); err == nil && fi.Size() > 0 {
		_ = zipFile(dbFile, archiveName)
	}
}

func zipFile(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.Base(source)
	header.Method = zip.Deflate

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}

	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	return err
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
				CreationFlags: 0x08000000,
			}
			_ = cmd.Start()
			return
		}
	}
}
