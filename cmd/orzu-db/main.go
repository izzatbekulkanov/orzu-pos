package main

import (
	"fmt"
	"log"
	"os"

	"github.com/izzatbekulkanov/orzu-pos/internal/database"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("   🐘 ORZU DB — Baza Diagnostika va Init v1.0    ")
	fmt.Println("   Namangan davlat pedagogika instituti RTTM      ")
	fmt.Println("==================================================")

	dbType := os.Getenv("ORZU_DB_TYPE")
	if dbType == "" {
		dbType = "sqlite"
	}

	fmt.Printf("Tekshirilayotgan rejim: %s\n", dbType)

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("❌ Ulanish muvaffaqiyatsiz bo'ldi: %v\n", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("SQL DB instansiyasini olib bo'lmadi: %v\n", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("❌ Ping xatosi: %v\n", err)
	}

	fmt.Println("✅ Baza aloqasi a'lo darajada!")
	fmt.Println("✅ Barcha jadvallar (Users, Products, Sales, Customers, Settings) to'liq tayyor.")
	fmt.Println("✅ Dastlabki sozlamalar va tovarlar muvaffaqiyatli yuklangan.")
}
