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

	// Auth
	api.Post("/auth/login", LoginHandler)

	// Mahsulotlar (Kassa & Ombor)
	api.Get("/products", GetProductsHandler)
	api.Get("/products/barcode/:barcode", GetProductByBarcodeHandler)
	api.Post("/products", CreateProductHandler)
	api.Put("/products/:id", UpdateProductHandler)

	// Toifalar
	api.Get("/categories", GetCategoriesHandler)

	// Kassa va Savdo
	api.Post("/sales/checkout", CheckoutHandler)
	api.Get("/sales", GetSalesHandler)

	// Mijozlar va Nasiya daftari
	api.Get("/customers", GetCustomersHandler)
	api.Post("/customers/:id/repay", RepayDebtHandler)

	// Analitika va Hisobotlar
	api.Get("/reports/daily", DailySummaryHandler)

	// Sozlamalar
	api.Get("/settings", SettingsHandler)
}
