package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/izzatbekulkanov/orzu-pos/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB - Ma'lumotlar bazasini ishga tushirish (PostgreSQL yoki SQLite)
func InitDB() (*gorm.DB, error) {
	dbType := os.Getenv("ORZU_DB_TYPE")
	dbURL := os.Getenv("ORZU_DB_URL")

	var dialector gorm.Dialector

	if dbType == "postgres" || (dbURL != "" && dbType != "sqlite") {
		if dbURL == "" {
			dbHost := getEnv("ORZU_DB_HOST", "localhost")
			dbPort := getEnv("ORZU_DB_PORT", "5432")
			dbUser := getEnv("ORZU_DB_USER", "postgres")
			dbPass := getEnv("ORZU_DB_PASSWORD", "postgres")
			dbName := getEnv("ORZU_DB_NAME", "orzu_pos")
			dbURL = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Tashkent",
				dbHost, dbUser, dbPass, dbName, dbPort)
		}
		log.Println("🐘 PostgreSQL bazasiga ulanilmoqda...")
		dialector = postgres.Open(dbURL)
	} else {
		// Standart holatda: Pure-Go SQLite (Oflayn va mustaqil rejim)
		dataDir := getEnv("ORZU_DATA_DIR", "data")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("data jildini yaratishda xato: %w", err)
		}
		dbPath := filepath.Join(dataDir, "orzu_pos.db")
		log.Printf("📁 SQLite oflayn bazasiga ulanilmoqda: %s\n", dbPath)
		dialector = sqlite.Open(dbPath)
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		// Agar PostgreSQL ulanmasa, xavfsiz holda SQLite ga fallback qilamiz
		if dbType == "postgres" {
			log.Printf("⚠️ PostgreSQL ulanmadi (%v). Oflayn SQLite bazasiga o'tilmoqda...\n", err)
			dataDir := getEnv("ORZU_DATA_DIR", "data")
			_ = os.MkdirAll(dataDir, 0755)
			dbPath := filepath.Join(dataDir, "orzu_pos.db")
			db, err = gorm.Open(sqlite.Open(dbPath), gormConfig)
			if err != nil {
				return nil, fmt.Errorf("fallback SQLite bazasini ochib bo'lmadi: %w", err)
			}
		} else {
			return nil, fmt.Errorf("ma'lumotlar bazasini ochib bo'lmadi: %w", err)
		}
	}

	// Migratsiyalarni amalga oshirish
	log.Println("🔄 Jadvallar migratsiyasi tekshirilmoqda...")
	err = db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Product{},
		&models.Customer{},
		&models.Sale{},
		&models.SaleItem{},
		&models.DebtRecord{},
		&models.Setting{},
	)
	if err != nil {
		return nil, fmt.Errorf("migratsiyada xato: %w", err)
	}

	DB = db
	log.Println("✅ Ma'lumotlar bazasi muvaffaqiyatli ishga tushdi!")

	// Boshlang'ich ma'lumotlarni tekshirish va ekish (Seed)
	seedInitialData(db)

	return db, nil
}

func seedInitialData(db *gorm.DB) {
	// 1. Sozlamalar
	defaultSettings := []models.Setting{
		{Key: "store_name", Value: "Olmos Orzu market", Description: "Do'kon nomi"},
		{Key: "store_address", Value: "Namangan viloyati", Description: "Manzil"},
		{Key: "store_phone", Value: "+998 90 000 00 00", Description: "Telefon raqam"},
		{Key: "receipt_header", Value: "Olmos Orzu marketiga xush kelibsiz!", Description: "Chek tepasidagi yozuv"},
		{Key: "receipt_footer", Value: "Xaridingiz uchun rahmat! Yana kutamiz!", Description: "Chek pastidagi yozuv"},
		{Key: "currency", Value: "so'm", Description: "Valyuta belgisi"},
		{Key: "tax_percent", Value: "0", Description: "QQS foizi"},
	}
	for _, s := range defaultSettings {
		var existing models.Setting
		if db.Where("key = ?", s.Key).First(&existing).Error != nil {
			db.Create(&s)
		}
	}

	// 2. Standart admin foydalanuvchisi
	var adminCount int64
	db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&adminCount)
	if adminCount == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := models.User{
			Username:     "admin",
			PasswordHash: string(hash),
			FullName:     "Bosh Administrator",
			Role:         models.RoleAdmin,
			PIN:          "1234",
			IsActive:     true,
		}
		db.Create(&admin)
		log.Println("👤 Standart admin yaratildi: login: admin, parol: admin123, PIN: 1234")
	}

	// 3. Standart kassir foydalanuvchisi
	var cashierCount int64
	db.Model(&models.User{}).Where("role = ?", models.RoleCashier).Count(&cashierCount)
	if cashierCount == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		cashier := models.User{
			Username:     "kassir1",
			PasswordHash: string(hash),
			FullName:     "Kassir 1",
			Role:         models.RoleCashier,
			PIN:          "0000",
			IsActive:     true,
		}
		db.Create(&cashier)
		log.Println("👤 Standart kassir yaratildi: login: kassir1, parol: 123456, PIN: 0000")
	}

	// 4. Standart demo kategoriyalar va mahsulotlar
	var prodCount int64
	db.Model(&models.Product{}).Count(&prodCount)
	if prodCount == 0 {
		catBread := models.Category{Name: "Non va qandolat", Icon: "croissant", Color: "#f59e0b"}
		catDrink := models.Category{Name: "Ichimliklar", Icon: "coffee", Color: "#3b82f6"}
		catDairy := models.Category{Name: "Sut mahsulotlari", Icon: "milk", Color: "#10b981"}
		catGrocery := models.Category{Name: "Oziq-ovqat / Baqollik", Icon: "shopping-bag", Color: "#8b5cf6"}
		db.Create(&catBread)
		db.Create(&catDrink)
		db.Create(&catDairy)
		db.Create(&catGrocery)

		demoProducts := []models.Product{
			{Barcode: "4780001234567", Name: "Samarqand noni (dona)", CategoryID: catBread.ID, BuyPrice: 3500, SellPrice: 5000, StockQuantity: 80, Unit: "dona"},
			{Barcode: "5449000000996", Name: "Coca-Cola 1.5L", CategoryID: catDrink.ID, BuyPrice: 11000, SellPrice: 14000, StockQuantity: 120, Unit: "dona"},
			{Barcode: "4780018890012", Name: "Nestle Sut 3.2% 1L", CategoryID: catDairy.ID, BuyPrice: 12500, SellPrice: 16000, StockQuantity: 50, Unit: "dona"},
			{Barcode: "4780029988771", Name: "Qand (oq shakar 1kg)", CategoryID: catGrocery.ID, BuyPrice: 9000, SellPrice: 11500, StockQuantity: 200, Unit: "kg"},
			{Barcode: "2000001002500", Name: "Olma Beshyulduz (kg)", CategoryID: catGrocery.ID, BuyPrice: 12000, SellPrice: 16000, StockQuantity: 45.5, Unit: "kg"},
		}
		for _, p := range demoProducts {
			db.Create(&p)
		}
		log.Println("🛒 Boshlang'ich demo tovarlar yuklandi.")
	}

	// 5. Standart mijoz (Oddiy xaridor)
	var custCount int64
	db.Model(&models.Customer{}).Count(&custCount)
	if custCount == 0 {
		cust := models.Customer{
			FullName: "Oddiy Xaridor",
			Phone:    "+998 00 000 00 00",
			Note:     "Standart xaridlar uchun umumiy mijoz",
		}
		db.Create(&cust)
	}
}

func getEnv(key, defVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defVal
}
