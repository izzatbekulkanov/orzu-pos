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
		log.Println("[PostgreSQL] Bazaga ulanilmoqda...")
		dialector = postgres.Open(dbURL)
	} else {
		// Standart holatda: Pure-Go SQLite (Oflayn va mustaqil rejim)
		dataDir := getEnv("ORZU_DATA_DIR", "data")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("data jildini yaratishda xato: %w", err)
		}
		dbPath := filepath.Join(dataDir, "orzu_pos.db")
		log.Printf("[SQLite] Oflayn bazaga ulanilmoqda: %s\n", dbPath)
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
			log.Printf("[PostgreSQL] Ulanmadi (%v). Oflayn SQLite bazasiga o'tilmoqda...\n", err)
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

	// SQLite uchun yuqori unumdorlik sozlamalari (WAL mode, xotira keshi, tezkor so'rovlar)
	if dbType != "postgres" {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.SetMaxOpenConns(1)
			sqlDB.SetMaxIdleConns(1)
			sqlDB.SetConnMaxLifetime(0)
		}
		db.Exec("PRAGMA journal_mode = WAL;")
		db.Exec("PRAGMA synchronous = NORMAL;")
		db.Exec("PRAGMA temp_store = MEMORY;")
		db.Exec("PRAGMA cache_size = -64000;")
		db.Exec("PRAGMA mmap_size = 30000000000;")
		db.Exec("PRAGMA busy_timeout = 5000;")
	}

	// Migratsiyalarni amalga oshirish
	log.Println("[Database] Jadvallar migratsiyasi tekshirilmoqda...")
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
	log.Println("[Database] Ma'lumotlar bazasi muvaffaqiyatli ishga tushdi.")

	// Boshlang'ich ma'lumotlarni tekshirish va ekish (Seed)
	seedInitialData(db)

	return db, nil
}

func seedInitialData(db *gorm.DB) {
	// 1. Sozlamalar
	defaultSettings := []models.Setting{
		{Key: "store_name", Value: "Olmos Orzu market", Description: "Do'kon nomi"},
		{Key: "company_name", Value: "Olmos Orzu Savdo MChJ", Description: "Tashkilot / Yuridik nomi"},
		{Key: "store_owner", Value: "Alisher Usmonov", Description: "Tizim egasi F.I.Sh."},
		{Key: "director_name", Value: "Rustam Karimov", Description: "Tashkilot rahbari F.I.Sh."},
		{Key: "store_address", Value: "Namangan viloyati, Yangiqo'rg'on tumani", Description: "Do'kon manzili"},
		{Key: "store_phone", Value: "+998 (90) 123-45-67", Description: "Telefon raqam"},
		{Key: "receipt_header", Value: "Olmos Orzu marketiga xush kelibsiz!", Description: "Chek tepasidagi yozuv"},
		{Key: "receipt_footer", Value: "Xaridingiz uchun rahmat! Yana kutamiz!", Description: "Chek pastidagi yozuv"},
		{Key: "currency", Value: "so'm", Description: "Valyuta belgisi"},
		{Key: "tax_percent", Value: "0", Description: "QQS foizi"},
		{Key: "label_width", Value: "80", Description: "Yorliq qog'ozi kengligi (58 yoki 80 mm)"},
		{Key: "label_autocut", Value: "true", Description: "Har bir yorliqdan keyin avtomatik qirqish"},
		{Key: "label_show_store", Value: "true", Description: "Yorliqda do'kon nomini chiqarish"},
		{Key: "label_show_price", Value: "true", Description: "Yorliqda narxni katta hajmda chiqarish"},
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
			Phone:        "+998 90 000 00 01",
			IsActive:     true,
		}
		db.Create(&admin)
		log.Println("[Database] Standart admin yaratildi: login: admin, parol: admin123, PIN: 1234")
	}

	// 3. Standart kassirlar (Ko'p kassirli tizim)
	var cashierCount int64
	db.Model(&models.User{}).Where("role = ?", models.RoleCashier).Count(&cashierCount)
	if cashierCount == 0 {
		hash1, _ := bcrypt.GenerateFromPassword([]byte("1111"), bcrypt.DefaultCost)
		kassir1 := models.User{
			Username:     "kassir1",
			PasswordHash: string(hash1),
			FullName:     "Dildora Karimova",
			Role:         models.RoleCashier,
			PIN:          "1111",
			Phone:        "+998 90 111 22 33",
			IsActive:     true,
		}
		db.Create(&kassir1)

		hash2, _ := bcrypt.GenerateFromPassword([]byte("2222"), bcrypt.DefaultCost)
		kassir2 := models.User{
			Username:     "kassir2",
			PasswordHash: string(hash2),
			FullName:     "Nodir Bekmurodov",
			Role:         models.RoleCashier,
			PIN:          "2222",
			Phone:        "+998 90 222 33 44",
			IsActive:     true,
		}
		db.Create(&kassir2)
		log.Println("[Database] Kassirlar yaratildi: Dildora Karimova (PIN: 1111), Nodir Bekmurodov (PIN: 2222)")
	}

	// 4. Standart toifalar va mahsulotlar (Faqat baza bo'm-bo'sh bo'lganda bir martagina yuklanadi)
	var catSeedDone models.Setting
	if db.Where("key = ?", "initial_categories_seeded").First(&catSeedDone).Error != nil {
		var catCount int64
		db.Unscoped().Model(&models.Category{}).Count(&catCount)
		if catCount == 0 {
			defaultCats := []models.Category{
				{Name: "Ichimlik suvi va Ichimliklar", Icon: "cup-soda", Color: "#0284c7"},
				{Name: "Oziq-ovqat va Baqollik", Icon: "shopping-bag", Color: "#d97706"},
				{Name: "Dorixona va Salomatlik", Icon: "heart-pulse", Color: "#e11d48"},
				{Name: "Non va Qandolat", Icon: "croissant", Color: "#f59e0b"},
				{Name: "Sut va Go'sht mahsulotlari", Icon: "milk", Color: "#10b981"},
				{Name: "Meva va Sabzavotlar", Icon: "apple", Color: "#84cc16"},
				{Name: "Maishiy kimyo", Icon: "sparkles", Color: "#8b5cf6"},
			}
			for _, c := range defaultCats {
				db.Create(&c)
			}
			log.Println("[Database] Boshlang'ich toifalar muvaffaqiyatli yuklandi.")
		}
		db.Create(&models.Setting{
			Key:         "initial_categories_seeded",
			Value:       "true",
			Description: "Boshlang'ich toifalar yuklanganlik holati",
		})
	}

	var prodSeedDone models.Setting
	if db.Where("key = ?", "initial_demo_products_seeded").First(&prodSeedDone).Error != nil {
		var prodCount int64
		db.Unscoped().Model(&models.Product{}).Count(&prodCount)
		if prodCount == 0 {
			var catBread, catDrink, catDairy, catGrocery models.Category
			db.Where("name LIKE ?", "%Non%").First(&catBread)
			db.Where("name LIKE ?", "%Ichimlik%").First(&catDrink)
			db.Where("name LIKE ?", "%Sut%").First(&catDairy)
			db.Where("name LIKE ?", "%Oziq%").First(&catGrocery)

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
			log.Println("[Database] Boshlang'ich demo tovarlar yuklandi.")
		}
		db.Create(&models.Setting{
			Key:         "initial_demo_products_seeded",
			Value:       "true",
			Description: "Boshlang'ich demo tovarlar yuklanganlik holati",
		})
	}

	// 4b. Ensure user's barcode 6927291699864 exists (Bir martagina tekshiriladi)
	var barcodeSeedDone models.Setting
	if db.Where("key = ?", "barcode_6927291699864_seeded").First(&barcodeSeedDone).Error != nil {
		var userProd models.Product
		if err := db.Unscoped().Where("barcode = ?", "6927291699864").First(&userProd).Error; err != nil {
			var firstCat models.Category
			db.First(&firstCat)
			catID := firstCat.ID
			if catID == 0 {
				catID = 1
			}
			db.Create(&models.Product{
				Barcode:       "6927291699864",
				ShortCode:     "1007",
				Name:          "Skanerlangan Mahsulot 6927291699864",
				CategoryID:    catID,
				Location:      "A-1 polka",
				BuyPrice:      8000,
				SellPrice:     12000,
				StockQuantity: 100,
				Unit:          "dona",
			})
		}
		db.Create(&models.Setting{
			Key:         "barcode_6927291699864_seeded",
			Value:       "true",
			Description: "Sinov shtrix kodi bir martalik holati",
		})
	}

	// 4c. Ensure all products have a 4-digit ShortCode
	var prodsWithoutCode []models.Product
	db.Where("short_code = '' OR short_code IS NULL").Find(&prodsWithoutCode)
	for i, p := range prodsWithoutCode {
		code := fmt.Sprintf("%04d", 1001+i)
		db.Model(&models.Product{}).Where("id = ?", p.ID).Update("short_code", code)
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
