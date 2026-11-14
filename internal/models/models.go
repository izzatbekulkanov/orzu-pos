package models

import (
	"time"

	"gorm.io/gorm"
)

// Role constants
const (
	RoleAdmin   = "admin"
	RoleCashier = "cashier"
	RoleManager = "manager"
)

// Payment methods
const (
	PaymentCash  = "cash"
	PaymentCard  = "card"
	PaymentQR    = "qr"
	PaymentDebt  = "debt"
	PaymentMixed = "mixed"
)

// User - Xodimlar va Administratorlar
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	FullName     string         `gorm:"size:128;not null" json:"full_name"`
	Role         string         `gorm:"size:32;default:'cashier'" json:"role"`
	PIN          string         `gorm:"size:8" json:"pin,omitempty"`
	Phone        string         `gorm:"size:32" json:"phone,omitempty"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Category - Tovar toifalari
type Category struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Icon      string         `gorm:"size:64" json:"icon"`
	Color     string         `gorm:"size:32" json:"color"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Products  []Product      `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

// Product - Mahsulotlar (Ombor va Kassa)
type Product struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Barcode       string         `gorm:"size:64;index;not null" json:"barcode"`
	ShortCode     string         `gorm:"size:16;index" json:"short_code"` // 4 xonali tezkor kod (masalan: 1001)
	Name          string         `gorm:"size:255;not null" json:"name"`
	CategoryID    uint           `gorm:"index" json:"category_id"`
	Category      *Category      `json:"category,omitempty"`
	BuyPrice      float64        `gorm:"type:decimal(14,2);default:0" json:"buy_price"`       // Tannarx
	SellPrice     float64        `gorm:"type:decimal(14,2);not null" json:"sell_price"`       // Sotish narxi
	StockQuantity float64        `gorm:"type:decimal(14,3);default:0" json:"stock_quantity"`  // Qoldiq (kg/dona)
	Unit          string         `gorm:"size:16;default:'dona'" json:"unit"`                  // dona, kg, litr, metr
	Location      string         `gorm:"size:128" json:"location"`                            // Tovar joylashuvi: polka / qator / muzlatgich
	ImageURL      string         `gorm:"type:text" json:"image_url"`                          // Tovar rasmi (Base64 yoki URL)
	MinStockAlert float64        `gorm:"default:5" json:"min_stock_alert"`
	ExpiryDate    *time.Time     `json:"expiry_date,omitempty"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// Customer - Mijozlar va Nasiya daftari
type Customer struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	FullName        string         `gorm:"size:128;not null" json:"full_name"`
	Phone           string         `gorm:"size:32;index" json:"phone"`
	DebtBalance     float64        `gorm:"type:decimal(14,2);default:0" json:"debt_balance"` // Qarzdorlik miqdori
	TotalPurchases  float64        `gorm:"type:decimal(14,2);default:0" json:"total_purchases"`
	DiscountPercent float64        `gorm:"default:0" json:"discount_percent"`
	Note            string         `gorm:"type:text" json:"note"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	DebtRecords     []DebtRecord   `gorm:"foreignKey:CustomerID" json:"debt_records,omitempty"`
}

// Sale - Savdo operatsiyalari (Cheklar)
type Sale struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	ReceiptNumber  string         `gorm:"size:64;uniqueIndex;not null" json:"receipt_number"`
	CashierID      uint           `gorm:"index" json:"cashier_id"`
	Cashier        *User          `json:"cashier,omitempty"`
	CustomerID     *uint          `gorm:"index" json:"customer_id,omitempty"`
	Customer       *Customer      `json:"customer,omitempty"`
	TotalAmount    float64        `gorm:"type:decimal(14,2);not null" json:"total_amount"`
	DiscountAmount float64        `gorm:"type:decimal(14,2);default:0" json:"discount_amount"`
	FinalAmount    float64        `gorm:"type:decimal(14,2);not null" json:"final_amount"`
	PaymentMethod  string         `gorm:"size:32;default:'cash'" json:"payment_method"`
	CashPaid       float64        `gorm:"type:decimal(14,2);default:0" json:"cash_paid"`
	CardPaid       float64        `gorm:"type:decimal(14,2);default:0" json:"card_paid"`
	QRPaid         float64        `gorm:"type:decimal(14,2);default:0" json:"qr_paid"`
	DebtAmount     float64        `gorm:"type:decimal(14,2);default:0" json:"debt_amount"`
	ChangeAmount   float64        `gorm:"type:decimal(14,2);default:0" json:"change_amount"`
	Status         string         `gorm:"size:32;default:'completed'" json:"status"` // completed, cancelled, refunded
	Note           string         `gorm:"size:255" json:"note"`
	CreatedAt      time.Time      `gorm:"index" json:"created_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Items          []SaleItem     `gorm:"foreignKey:SaleID" json:"items"`
}

// SaleItem - Chekdagi har bir tovar qatori
type SaleItem struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	SaleID      uint           `gorm:"index;not null" json:"sale_id"`
	ProductID   uint           `gorm:"index;not null" json:"product_id"`
	Barcode     string         `gorm:"size:64" json:"barcode"`
	ProductName string         `gorm:"size:255;not null" json:"product_name"`
	UnitPrice   float64        `gorm:"type:decimal(14,2);not null" json:"unit_price"`
	Quantity    float64        `gorm:"type:decimal(14,3);not null" json:"quantity"`
	Unit        string         `gorm:"size:16;default:'dona'" json:"unit"`
	TotalPrice  float64        `gorm:"type:decimal(14,2);not null" json:"total_price"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// DebtRecord - Nasiya daftari harakatlari (Qarz olish va Qarz to'lash)
type DebtRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CustomerID   uint      `gorm:"index;not null" json:"customer_id"`
	SaleID       *uint     `gorm:"index" json:"sale_id,omitempty"`
	Type         string    `gorm:"size:16;not null" json:"type"` // 'borrow' (qarz oldi) yoki 'repay' (qarz to'ladi)
	Amount       float64   `gorm:"type:decimal(14,2);not null" json:"amount"`
	BalanceAfter float64   `gorm:"type:decimal(14,2);not null" json:"balance_after"`
	Note         string    `gorm:"size:255" json:"note"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

// Setting - Do'kon sozlamalari
type Setting struct {
	Key         string `gorm:"size:64;primaryKey" json:"key"`
	Value       string `gorm:"type:text;not null" json:"value"`
	Description string `gorm:"size:255" json:"description"`
}
