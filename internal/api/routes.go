package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// SetupRoutes - Barcha API marshrutlarini sozlash
func SetupRoutes(app *fiber.App) {
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	api := app.Group("/api")

	// Auth (Admin va Kassir)
	api.Post("/auth/login", LoginHandler)
	api.Post("/auth/cashier-login", CashierLoginHandler)

	// Kassirlar boshqaruvi
	api.Get("/cashiers", GetCashiersHandler)
	api.Post("/cashiers", CreateCashierHandler)
	api.Put("/cashiers/:id", UpdateCashierHandler)
	api.Delete("/cashiers/:id", DeleteCashierHandler)

	// Mahsulotlar (Kassa & Ombor)
	api.Get("/products", GetProductsHandler)
	api.Get("/products/barcode/:barcode", GetProductByBarcodeHandler)
	api.Post("/products", CreateProductHandler)
	api.Put("/products/:id", UpdateProductHandler)
	api.Post("/products/:id/stock", AddProductStockHandler)
	api.Delete("/products/:id", DeleteProductHandler)

	// Toifalar
	api.Get("/categories", GetCategoriesHandler)
	api.Post("/categories", CreateCategoryHandler)
	api.Put("/categories/:id", UpdateCategoryHandler)
	api.Delete("/categories/:id", DeleteCategoryHandler)

	// Kassa va Savdo
	api.Post("/sales/checkout", CheckoutHandler)
	api.Get("/sales", GetSalesHandler)

	// Mijozlar va Nasiya daftari
	api.Get("/customers", GetCustomersHandler)
	api.Post("/customers", CreateCustomerHandler)
	api.Put("/customers/:id", UpdateCustomerHandler)
	api.Delete("/customers/:id", DeleteCustomerHandler)
	api.Post("/customers/:id/repay", RepayDebtHandler)

	// Analitika va Hisobotlar
	api.Get("/reports/daily", DailySummaryHandler)

	// Sozlamalar (Tashkilot, rekvizitlar, yorliq o'lchamlari)
	api.Get("/settings", SettingsHandler)
	api.Post("/settings", UpdateSettingsHandler)

	// Ma'lumotlar bazasi (DB boshqaruvi)
	api.Get("/db/stats", DBStatsHandler)
	api.Post("/db/vacuum", DBVacuumHandler)

	// Litsenziya va Faollashtirish
	api.Get("/license", LicenseHandler)
	api.Post("/license/activate", ActivateHandler)

	// Ilovalar o'rtasida o'tish (Standalone Desktop rejimida) va Faollik holati
	api.Post("/open-app", OpenAppHandler)
	api.Post("/heartbeat/:app", HeartbeatHandler)
	api.Post("/heartbeat-leave/:app", HeartbeatLeaveHandler)
	api.Get("/apps-status", AppStatusHandler)

	// Chek printeri va Yorliq chop etish (ESC/POS RAW)
	api.Post("/printer/print-receipt", PrintReceiptHandler)
	api.Post("/printer/print-label", PrintLabelHandler)
	api.Get("/printer/status", PrinterStatusHandler)

	// AI Tovar Rasmi Generatsiyasi (OpenRouter AI)
	api.Post("/ai/generate-product-image", GenerateProductImageHandler)

	// Tizim Yangilanishlari (GitHub Auto-Updater)
	api.Get("/updates/check", CheckUpdatesHandler)
	api.Post("/updates/apply", ApplyUpdateHandler)
}
