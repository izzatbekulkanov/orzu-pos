package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/izzatbekulkanov/orzu-pos/internal/database"
	"github.com/izzatbekulkanov/orzu-pos/internal/models"
	"gorm.io/gorm"
)

type CartItemInput struct {
	ProductID uint    `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type CheckoutInput struct {
	CashierID      uint            `json:"cashier_id"`
	CustomerID     *uint           `json:"customer_id,omitempty"`
	Items          []CartItemInput `json:"items"`
	PaymentMethod  string          `json:"payment_method"` // cash, card, qr, debt, mixed
	CashPaid       float64         `json:"cash_paid"`
	CardPaid       float64         `json:"card_paid"`
	QRPaid         float64         `json:"qr_paid"`
	DebtAmount     float64         `json:"debt_amount"`
	DiscountAmount float64         `json:"discount_amount"`
	Note           string          `json:"note"`
}

type DailySummary struct {
	TotalSalesCount int64   `json:"total_sales_count"`
	TotalRevenue    float64 `json:"total_revenue"`
	TotalCash       float64 `json:"total_cash"`
	TotalCard       float64 `json:"total_card"`
	TotalQR         float64 `json:"total_qr"`
	TotalDebt       float64 `json:"total_debt"`
	TotalProfit     float64 `json:"total_profit"`
}

// ProcessSale - Savdoni tranzaksiya ichida atomik amalga oshirish
func ProcessSale(input CheckoutInput) (*models.Sale, string, error) {
	if len(input.Items) == 0 {
		return nil, "", fmt.Errorf("savdo savatida tovarlar yo'q")
	}

	var createdSale *models.Sale
	var receiptText string

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var totalAmount float64
		var totalBuyCost float64
		var saleItems []models.SaleItem

		// 1. Tovar qoldiqlarini tekshirish va hisoblash
		for _, itemInput := range input.Items {
			if itemInput.Quantity <= 0 {
				return fmt.Errorf("tovar miqdori 0 dan katta bo'lishi kerak")
			}

			var product models.Product
			if err := tx.First(&product, itemInput.ProductID).Error; err != nil {
				return fmt.Errorf("mahsulot topilmadi (ID: %d): %w", itemInput.ProductID, err)
			}

			unitPrice := itemInput.UnitPrice
			if unitPrice <= 0 {
				unitPrice = product.SellPrice
			}
			lineTotal := unitPrice * itemInput.Quantity
			totalAmount += lineTotal
			totalBuyCost += product.BuyPrice * itemInput.Quantity

			// Ombor qoldig'ini qat'iy tekshirish - ortiqcha sotishga yo'l qo'yilmaydi!
			if product.StockQuantity < itemInput.Quantity {
				return fmt.Errorf("omborda '%s' mahsulotidan yetarli qoldiq mavjud emas! Qoldiq: %.2f %s, so'ralgan: %.2f %s",
					product.Name, product.StockQuantity, product.Unit, itemInput.Quantity, product.Unit)
			}

			// Ombor qoldig'ini kamaytirish
			product.StockQuantity -= itemInput.Quantity
			if err := tx.Save(&product).Error; err != nil {
				return fmt.Errorf("ombor qoldig'ini yangilashda xato: %w", err)
			}

			saleItems = append(saleItems, models.SaleItem{
				ProductID:   product.ID,
				Barcode:     product.Barcode,
				ProductName: product.Name,
				UnitPrice:   unitPrice,
				Quantity:    itemInput.Quantity,
				Unit:        product.Unit,
				TotalPrice:  lineTotal,
			})
		}

		finalAmount := totalAmount - input.DiscountAmount
		if finalAmount < 0 {
			finalAmount = 0
		}

		// To'lov hisob-kitobi
		var changeAmount float64
		switch input.PaymentMethod {
		case models.PaymentCash:
			if input.CashPaid <= 0 {
				input.CashPaid = finalAmount
			}
			if input.CashPaid >= finalAmount {
				changeAmount = input.CashPaid - finalAmount
			}
		case models.PaymentCard:
			if input.CardPaid <= 0 {
				input.CardPaid = finalAmount
			}
		case models.PaymentQR:
			if input.QRPaid <= 0 {
				input.QRPaid = finalAmount
			}
		case models.PaymentDebt:
			if input.DebtAmount <= 0 {
				input.DebtAmount = finalAmount
			}
		case models.PaymentMixed:
			totalTendered := input.CashPaid + input.CardPaid + input.QRPaid + input.DebtAmount
			if totalTendered >= finalAmount {
				changeAmount = totalTendered - finalAmount
			}
		default:
			if input.CashPaid <= 0 {
				input.CashPaid = finalAmount
			}
		}

		// Chek raqamini generatsiya qilish (masalan ORZU-20261113-0001)
		now := time.Now()
		datePrefix := now.Format("20060102")
		var countToday int64
		tx.Model(&models.Sale{}).Where("DATE(created_at) = DATE(?)", now).Count(&countToday)
		receiptNumber := fmt.Sprintf("ORZU-%s-%04d", datePrefix, countToday+1)

		sale := models.Sale{
			ReceiptNumber:  receiptNumber,
			CashierID:      input.CashierID,
			CustomerID:     input.CustomerID,
			TotalAmount:    totalAmount,
			DiscountAmount: input.DiscountAmount,
			FinalAmount:    finalAmount,
			PaymentMethod:  input.PaymentMethod,
			CashPaid:       input.CashPaid,
			CardPaid:       input.CardPaid,
			QRPaid:         input.QRPaid,
			DebtAmount:     input.DebtAmount,
			ChangeAmount:   changeAmount,
			Status:         "completed",
			Note:           input.Note,
			CreatedAt:      now,
			Items:          saleItems,
		}

		if err := tx.Create(&sale).Error; err != nil {
			return fmt.Errorf("savdo yozuvini yaratishda xato: %w", err)
		}

		// Nasiya (Qarz) bo'lsa mijoz balansini oshirish
		if input.DebtAmount > 0 && input.CustomerID != nil {
			var customer models.Customer
			if err := tx.First(&customer, *input.CustomerID).Error; err != nil {
				return fmt.Errorf("mijoz topilmadi: %w", err)
			}
			customer.DebtBalance += input.DebtAmount
			customer.TotalPurchases += finalAmount
			if err := tx.Save(&customer).Error; err != nil {
				return fmt.Errorf("mijoz balansini yangilashda xato: %w", err)
			}

			// Nasiya daftari qaydi
			debtRecord := models.DebtRecord{
				CustomerID:   customer.ID,
				SaleID:       &sale.ID,
				Type:         "borrow",
				Amount:       input.DebtAmount,
				BalanceAfter: customer.DebtBalance,
				Note:         fmt.Sprintf("Chek: %s bo'yicha xarid", receiptNumber),
				CreatedAt:    now,
			}
			if err := tx.Create(&debtRecord).Error; err != nil {
				return fmt.Errorf("qarz qaydini saqlashda xato: %w", err)
			}
		}

		createdSale = &sale
		receiptText = generateReceiptText(&sale)
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	return createdSale, receiptText, nil
}

// RepayDebt - Mijoz qarzini to'lash
func RepayDebt(customerID uint, amount float64, note string) (*models.DebtRecord, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("to'lov summasi 0 dan katta bo'lishi kerak")
	}

	var record *models.DebtRecord
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var customer models.Customer
		if err := tx.First(&customer, customerID).Error; err != nil {
			return fmt.Errorf("mijoz topilmadi: %w", err)
		}

		customer.DebtBalance -= amount
		if err := tx.Save(&customer).Error; err != nil {
			return err
		}

		rec := models.DebtRecord{
			CustomerID:   customer.ID,
			Type:         "repay",
			Amount:       amount,
			BalanceAfter: customer.DebtBalance,
			Note:         note,
			CreatedAt:    time.Now(),
		}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		record = &rec
		return nil
	})

	return record, err
}

// GetDailySummary - Kunlik statistika
func GetDailySummary(date time.Time) (*DailySummary, error) {
	var summary DailySummary

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var sales []models.Sale
	err := database.DB.Preload("Items").Where("created_at >= ? AND created_at < ? AND status = ?", startOfDay, endOfDay, "completed").Find(&sales).Error
	if err != nil {
		return nil, err
	}

	summary.TotalSalesCount = int64(len(sales))
	for _, s := range sales {
		summary.TotalRevenue += s.FinalAmount

		cash := s.CashPaid
		card := s.CardPaid
		qr := s.QRPaid
		debt := s.DebtAmount

		// Agar oldingi cheklarda tender summasi 0 qolgan bo'lsa, to'lov usuliga qarab to'ldirish
		if cash == 0 && card == 0 && qr == 0 && debt == 0 {
			switch s.PaymentMethod {
			case models.PaymentCard:
				card = s.FinalAmount
			case models.PaymentQR:
				qr = s.FinalAmount
			case models.PaymentDebt:
				debt = s.FinalAmount
			default:
				cash = s.FinalAmount
			}
		}

		summary.TotalCash += cash
		summary.TotalCard += card
		summary.TotalQR += qr
		summary.TotalDebt += debt

		// Foydani hisoblash
		for _, item := range s.Items {
			var p models.Product
			if err := database.DB.Select("buy_price").First(&p, item.ProductID).Error; err == nil {
				profitPerUnit := item.UnitPrice - p.BuyPrice
				summary.TotalProfit += profitPerUnit * item.Quantity
			}
		}
	}

	return &summary, nil
}

// generateReceiptText - ESC/POS termal printer chek matnini generatsiya qilish
func generateReceiptText(sale *models.Sale) string {
	var sb strings.Builder
	storeName := "Olmos Orzu market"
	var setting models.Setting
	if err := database.DB.Where("key = ?", "store_name").First(&setting).Error; err == nil && setting.Value != "" {
		storeName = setting.Value
	}

	sep := strings.Repeat("-", 32)
	sb.WriteString(fmt.Sprintf("%s\n", centerText(storeName, 32)))
	sb.WriteString(fmt.Sprintf("%s\n", centerText("Xush kelibsiz!", 32)))
	sb.WriteString(sep + "\n")
	sb.WriteString(fmt.Sprintf("Chek: %s\n", sale.ReceiptNumber))
	sb.WriteString(fmt.Sprintf("Sana: %s\n", sale.CreatedAt.Format("02.01.2006 15:04:05")))
	sb.WriteString(sep + "\n")

	for i, item := range sale.Items {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.ProductName))
		qtyStr := fmt.Sprintf("%.3f", item.Quantity)
		qtyStr = strings.TrimRight(strings.TrimRight(qtyStr, "0"), ".")
		sb.WriteString(fmt.Sprintf("   %s %s x %.0f = %.0f so'm\n",
			qtyStr, item.Unit, item.UnitPrice, item.TotalPrice))
	}

	sb.WriteString(sep + "\n")
	sb.WriteString(fmt.Sprintf("Jami:        %14.0f so'm\n", sale.TotalAmount))
	if sale.DiscountAmount > 0 {
		sb.WriteString(fmt.Sprintf("Chegirma:    %14.0f so'm\n", sale.DiscountAmount))
	}
	sb.WriteString(fmt.Sprintf("TO'LANISHI:  %14.0f so'm\n", sale.FinalAmount))
	sb.WriteString(fmt.Sprintf("To'lov turi: %14s\n", sale.PaymentMethod))
	if sale.ChangeAmount > 0 {
		sb.WriteString(fmt.Sprintf("Qaytim:      %14.0f so'm\n", sale.ChangeAmount))
	}
	sb.WriteString(sep + "\n")
	sb.WriteString(fmt.Sprintf("%s\n", centerText("Xaridingiz uchun rahmat!", 32)))

	return sb.String()
}

func centerText(s string, width int) string {
	if len(s) >= width {
		return s
	}
	pad := (width - len(s)) / 2
	return strings.Repeat(" ", pad) + s
}
