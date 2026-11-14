package services

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type BackupFile struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	SizeKB    string `json:"size_kb"`
	CreatedAt string `json:"created_at"`
}

func getBackupDir() string {
	candidates := []string{
		`D:\ORZU_POS\backups`,
		"backups",
	}
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return p
		}
	}
	_ = os.MkdirAll("backups", 0755)
	return "backups"
}

func ListBackups() ([]BackupFile, error) {
	dir := getBackupDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []BackupFile{}, nil
	}

	var list []BackupFile
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".zip" {
			info, err := e.Info()
			if err != nil {
				continue
			}
			list = append(list, BackupFile{
				Name:      e.Name(),
				Path:      filepath.Join(dir, e.Name()),
				SizeKB:    fmt.Sprintf("%.1f KB", float64(info.Size())/1024),
				CreatedAt: info.ModTime().Format("02.01.2006 15:04:05"),
			})
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt > list[j].CreatedAt
	})

	return list, nil
}

func CreateBackup() (*BackupFile, error) {
	dir := getBackupDir()
	_ = os.MkdirAll(dir, 0755)

	dbCandidates := []string{
		`D:\ORZU_POS\data\orzu_pos.db`,
		filepath.Join("data", "orzu_pos.db"),
	}

	var foundDB string
	for _, p := range dbCandidates {
		if _, err := os.Stat(p); err == nil {
			foundDB = p
			break
		}
	}

	if foundDB == "" {
		return nil, fmt.Errorf("orzu_pos.db fayli topilmadi")
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	fileName := fmt.Sprintf("orzu_pos_backup_%s.zip", timestamp)
	targetZip := filepath.Join(dir, fileName)

	zipfile, err := os.Create(targetZip)
	if err != nil {
		return nil, err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	info, err := os.Stat(foundDB)
	if err != nil {
		return nil, err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return nil, err
	}
	header.Name = filepath.Base(foundDB)
	header.Method = zip.Deflate

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(foundDB)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	if err != nil {
		return nil, err
	}

	fi, _ := os.Stat(targetZip)
	return &BackupFile{
		Name:      fileName,
		Path:      targetZip,
		SizeKB:    fmt.Sprintf("%.1f KB", float64(fi.Size())/1024),
		CreatedAt: time.Now().Format("02.01.2006 15:04:05"),
	}, nil
}
