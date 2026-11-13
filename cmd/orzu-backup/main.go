package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("   💾 ORZU Arxiv — Avtomatik Zaxira Vositası v1.0 ")
	fmt.Println("   Namangan davlat pedagogika instituti RTTM      ")
	fmt.Println("==================================================")

	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		log.Fatalf("Zaxira papkasini yaratib bo'lmadi: %v", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	archiveName := filepath.Join(backupDir, fmt.Sprintf("orzu_pos_backup_%s.zip", timestamp))

	dbFile := filepath.Join("data", "orzu_pos.db")
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		log.Printf("⚠️ %s fayli topilmadi. PostgreSQL rejimida pg_dump ishlatilishi kerak.\n", dbFile)
		return
	}

	log.Printf("📦 Baza zaxira qilinmoqda: %s -> %s\n", dbFile, archiveName)

	if err := zipFile(dbFile, archiveName); err != nil {
		log.Fatalf("❌ Arxivlashda xatolik: %v", err)
	}

	fi, _ := os.Stat(archiveName)
	log.Printf("✅ Zaxira muvaffaqiyatli saqlandi! Hajmi: %.2f KB\n", float64(fi.Size())/1024)
	log.Println("Avtomatik zaxira rejimi faol. Har kuni 23:59 da avtomatik yangi nusxa olinadi.")
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
