package api

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/izzatbekulkanov/orzu-pos/internal/database"
	"github.com/izzatbekulkanov/orzu-pos/internal/models"
	"github.com/izzatbekulkanov/orzu-pos/internal/services"
	"golang.org/x/crypto/bcrypt"
)

// LoginHandler - Tizimga kirish (Parol yoki PIN orqali)
func LoginHandler(c *fiber.Ctx) error {
	type LoginInput struct {
		Username string `json:"username"`
		Password string `json:"password"`
		PIN      string `json:"pin"`
	}
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Noto'g'ri so'rov formati"})
	}

	var user models.User
	if input.PIN != "" {
		if err := database.DB.Where("pin = ? AND is_active = ?", input.PIN, true).First(&user).Error; err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "PIN kod noto'g'ri"})
		}
	} else {
		if err := database.DB.Where("username = ? AND is_active = ?", input.Username, true).First(&user).Error; err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "Login yoki parol noto'g'ri"})
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "Login yoki parol noto'g'ri"})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user": fiber.Map{
			"id":        user.ID,
			"username":  user.Username,
			"full_name": user.FullName,
			"role":      user.Role,
		},
	})
}

// GetProductsHandler - Tovarlarni qidirish va ro'yxatini olish
func GetProductsHandler(c *fiber.Ctx) error {
	q := c.Query("q")
	catID := c.Query("category_id")

	db := database.DB.Preload("Category").Where("is_active = ?", true)

	if q != "" {
		qLike := "%" + strings.ToLower(q) + "%"
		db = db.Where("LOWER(name) LIKE ? OR barcode LIKE ?", qLike, "%"+q+"%")
	}
	if catID != "" {
		db = db.Where("category_id = ?", catID)
	}

	var products []models.Product
	if err := db.Order("name ASC").Find(&products).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(products)
}

// GetProductByBarcodeHandler - Shtrix-kod orqali bir zumda tovarni topish (Kassa uchun)
func GetProductByBarcodeHandler(c *fiber.Ctx) error {
	barcode := c.Params("barcode")
	if barcode == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Shtrix-kod kiritilmadi"})
	}

	var product models.Product
	if err := database.DB.Preload("Category").Where("barcode = ? AND is_active = ?", barcode, true).First(&product).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Bunday shtrix-kodli tovar topilmadi"})
	}

	return c.JSON(product)
}

// CreateProductHandler - Yangi tovar qo'shish (Ombor kirim)
func CreateProductHandler(c *fiber.Ctx) error {
	var product models.Product
	if err := c.BodyParser(&product); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if product.Barcode == "" || product.Name == "" || product.SellPrice <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Shtrix-kod, tovar nomi va sotish narxi kiritilishi shart"})
	}

	if err := database.DB.Create(&product).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(product)
}

// UpdateProductHandler - Tovarni tahrirlash
func UpdateProductHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Tovar topilmadi"})
	}

	if err := c.BodyParser(&product); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	if err := database.DB.Save(&product).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(product)
}

// GetCategoriesHandler - Kategoriyalar
func GetCategoriesHandler(c *fiber.Ctx) error {
	var categories []models.Category
	if err := database.DB.Order("name ASC").Find(&categories).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(categories)
}

// CheckoutHandler - Savdoni yakunlash va chek chiqarish
func CheckoutHandler(c *fiber.Ctx) error {
	var input services.CheckoutInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Noto'g'ri ma'lumotlar formati: " + err.Error()})
	}

	sale, receiptText, err := services.ProcessSale(input)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"sale":         sale,
		"receipt_text": receiptText,
	})
}

// GetSalesHandler - Sotuvlar ro'yxati va tarixi
func GetSalesHandler(c *fiber.Ctx) error {
	limitStr := c.Query("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	var sales []models.Sale
	if err := database.DB.Preload("Items").Preload("Customer").Preload("Cashier").
		Order("created_at DESC").Limit(limit).Find(&sales).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(sales)
}

// GetCustomersHandler - Mijozlar va nasiyadorlar
func GetCustomersHandler(c *fiber.Ctx) error {
	q := c.Query("q")
	db := database.DB.Model(&models.Customer{})
	if q != "" {
		qLike := "%" + strings.ToLower(q) + "%"
		db = db.Where("LOWER(full_name) LIKE ? OR phone LIKE ?", qLike, "%"+q+"%")
	}

	var customers []models.Customer
	if err := db.Order("full_name ASC").Find(&customers).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(customers)
}

// RepayDebtHandler - Qarz to'lash
func RepayDebtHandler(c *fiber.Ctx) error {
	custID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Noto'g'ri mijoz ID"})
	}

	type RepayInput struct {
		Amount float64 `json:"amount"`
		Note   string  `json:"note"`
	}
	var input RepayInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	record, err := services.RepayDebt(uint(custID), input.Amount, input.Note)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "record": record})
}

// DailySummaryHandler - Kunlik tushum va hisobot
func DailySummaryHandler(c *fiber.Ctx) error {
	summary, err := services.GetDailySummary(time.Now())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(summary)
}

// SettingsHandler - Sozlamalar
func SettingsHandler(c *fiber.Ctx) error {
	var settings []models.Setting
	if err := database.DB.Find(&settings).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	res := make(map[string]string)
	for _, s := range settings {
		res[s.Key] = s.Value
	}
	return c.JSON(res)
}
