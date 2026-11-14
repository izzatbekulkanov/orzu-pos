//go:build !windows

package printer

import (
	"fmt"
	"strings"
)

type ReceiptItem struct {
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	Unit       string  `json:"unit"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
}

type ReceiptRequest struct {
	StoreName     string        `json:"store_name"`
	CompanyName   string        `json:"company_name"`
	StoreAddress  string        `json:"store_address"`
	StorePhone    string        `json:"store_phone"`
	DirectorName  string        `json:"director_name"`
	ReceiptNo     string        `json:"receipt_no"`
	Date          string        `json:"date"`
	Reprint       bool          `json:"reprint"`
	ReprintDate   string        `json:"reprint_date"`
	CashierName   string        `json:"cashier_name"`
	PaymentMethod string        `json:"payment_method"`
	CustomerName  string        `json:"customer_name"`
	TotalAmount   float64       `json:"total_amount"`
	CashGiven     float64       `json:"cash_given"`
	Change        float64       `json:"change"`
	Items         []ReceiptItem `json:"items"`
}

type LabelRequest struct {
	StoreName  string  `json:"store_name"`
	Name       string  `json:"name"`
	Barcode    string  `json:"barcode"`
	ShortCode  string  `json:"short_code"`
	Price      float64 `json:"price"`
	Unit       string  `json:"unit"`
	Copies     int     `json:"copies"`
	LabelWidth string  `json:"label_width"`
	AutoCut    bool    `json:"auto_cut"`
	ShowStore  bool    `json:"show_store"`
	ShowPrice  bool    `json:"show_price"`
}

func FormatMoney(val float64) string {
	n := int64(val)
	if n < 0 {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var res []string
	for len(s) > 3 {
		res = append([]string{s[len(s)-3:]}, res...)
		s = s[:len(s)-3]
	}
	if len(s) > 0 {
		res = append([]string{s}, res...)
	}
	return strings.Join(res, " ")
}

func GetDefaultPrinter() (string, error) {
	return "Generic Thermal Printer", nil
}

func PrintReceipt(req ReceiptRequest) (string, error) {
	return "Generic Thermal Printer", fmt.Errorf("direct print only supported on Windows")
}

func PrintLabel(req LabelRequest) (string, error) {
	return "Generic Thermal Printer", fmt.Errorf("direct label print only supported on Windows")
}
