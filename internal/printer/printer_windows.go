//go:build windows

package printer

import (
	"bytes"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	modWinspool           = syscall.NewLazyDLL("winspool.drv")
	procOpenPrinterW      = modWinspool.NewProc("OpenPrinterW")
	procStartDocPrinterW  = modWinspool.NewProc("StartDocPrinterW")
	procStartPagePrinter  = modWinspool.NewProc("StartPagePrinter")
	procWritePrinter      = modWinspool.NewProc("WritePrinter")
	procEndPagePrinter    = modWinspool.NewProc("EndPagePrinter")
	procEndDocPrinter     = modWinspool.NewProc("EndDocPrinter")
	procClosePrinter      = modWinspool.NewProc("ClosePrinter")
	procGetDefaultPrinter = modWinspool.NewProc("GetDefaultPrinterW")
)

type DOC_INFO_1 struct {
	DocName    *uint16
	OutputFile *uint16
	Datatype   *uint16
}

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

// GetDefaultPrinter returns the name of default printer in Windows
func GetDefaultPrinter() (string, error) {
	var size uint32 = 0
	procGetDefaultPrinter.Call(0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return "", fmt.Errorf("standart printer topilmadi")
	}
	buf := make([]uint16, size)
	r1, _, err := procGetDefaultPrinter.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r1 == 0 {
		return "", err
	}
	return syscall.UTF16ToString(buf), nil
}

// PrintRawBytes sends raw bytes directly to the given Windows printer spooler
func PrintRawBytes(printerName string, docName string, data []byte) error {
	var hPrinter uintptr
	pNamePtr, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		return err
	}

	r1, _, err := procOpenPrinterW.Call(
		uintptr(unsafe.Pointer(pNamePtr)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if r1 == 0 {
		return fmt.Errorf("printerga ulanishda xato (%s): %v", printerName, err)
	}
	defer procClosePrinter.Call(hPrinter)

	dNamePtr, _ := syscall.UTF16PtrFromString(docName)
	rawPtr, _ := syscall.UTF16PtrFromString("RAW")

	docInfo := DOC_INFO_1{
		DocName:    dNamePtr,
		OutputFile: nil,
		Datatype:   rawPtr,
	}

	jobId, _, err := procStartDocPrinterW.Call(
		hPrinter,
		1,
		uintptr(unsafe.Pointer(&docInfo)),
	)
	if jobId == 0 {
		return fmt.Errorf("StartDocPrinter error: %v", err)
	}
	defer procEndDocPrinter.Call(hPrinter)

	r1, _, err = procStartPagePrinter.Call(hPrinter)
	if r1 == 0 {
		return fmt.Errorf("StartPagePrinter error: %v", err)
	}
	defer procEndPagePrinter.Call(hPrinter)

	var written uint32
	r1, _, err = procWritePrinter.Call(
		hPrinter,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&written)),
	)
	if r1 == 0 {
		return fmt.Errorf("WritePrinter error: %v", err)
	}

	return nil
}

// FormatMoney formats numbers like 120000 -> "120 000"
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

// BuildEscPosReceipt constructs standard thermal receipt bytes with ESC/POS commands
func BuildEscPosReceipt(req ReceiptRequest) []byte {
	var buf bytes.Buffer

	// 1. Initialize printer
	buf.Write([]byte("\x1b\x40"))

	// 2. Open Cash Drawer (pulse pin 2)
	buf.Write([]byte("\x1b\x70\x00\x19\xfa"))

	// 3. Header: Store Name (Center, Double Size, Bold)
	storeTitle := strings.TrimSpace(req.StoreName)
	if storeTitle == "" {
		storeTitle = "OLMOS ORZU MARKET"
	}
	buf.Write([]byte("\x1b\x61\x01")) // Center
	buf.Write([]byte("\x1b\x45\x01")) // Bold ON
	buf.Write([]byte("\x1d\x21\x11")) // Double Width & Height
	buf.WriteString(strings.ToUpper(storeTitle) + "\n")
	buf.Write([]byte("\x1d\x21\x00")) // Normal size
	buf.Write([]byte("\x1b\x45\x00")) // Bold OFF

	if req.CompanyName != "" && req.CompanyName != storeTitle {
		buf.WriteString(req.CompanyName + "\n")
	}
	if req.StoreAddress != "" {
		buf.WriteString(req.StoreAddress + "\n")
	}
	phone := req.StorePhone
	if phone == "" {
		phone = "+998 (90) 123-45-67"
	}
	buf.WriteString(fmt.Sprintf("Tel: %s\n", phone))
	buf.WriteString("================================\n")

	if req.Reprint {
		buf.Write([]byte("\x1b\x61\x01")) // Center
		buf.Write([]byte("\x1b\x45\x01")) // Bold ON
		buf.WriteString("*** QAYTA CHOP ETILGAN NUSXA ***\n")
		buf.Write([]byte("\x1b\x45\x00")) // Bold OFF
		buf.WriteString("================================\n")
	}

	// 4. Receipt details (Left aligned)
	buf.Write([]byte("\x1b\x61\x00")) // Left align
	dateStr := req.Date
	if dateStr == "" {
		dateStr = time.Now().Format("02.01.2006 15:04:05")
	}
	buf.WriteString(fmt.Sprintf("Chek raqami:    %s\n", req.ReceiptNo))
	if req.Reprint {
		buf.WriteString(fmt.Sprintf("Asl sana:       %s\n", dateStr))
		reprintDate := req.ReprintDate
		if reprintDate == "" {
			reprintDate = time.Now().Format("02.01.2006 15:04:05")
		}
		buf.WriteString(fmt.Sprintf("Qayta chop:     %s\n", reprintDate))
	} else {
		buf.WriteString(fmt.Sprintf("Sana:           %s\n", dateStr))
	}
	if req.CashierName != "" {
		buf.WriteString(fmt.Sprintf("Kassir:         %s\n", req.CashierName))
	}
	payMethodUz := "NAQD PUL"
	switch strings.ToLower(req.PaymentMethod) {
	case "card":
		payMethodUz = "BANK KARTASI"
	case "qr":
		payMethodUz = "CLICK / PAYME (QR)"
	case "debt":
		payMethodUz = "NASIYA (QARZ)"
	}
	buf.WriteString(fmt.Sprintf("To'lov turi:    %s\n", payMethodUz))
	if req.CustomerName != "" {
		buf.WriteString(fmt.Sprintf("Mijoz:          %s\n", req.CustomerName))
	}
	buf.WriteString("--------------------------------\n")

	// 5. Items
	for i, item := range req.Items {
		unit := item.Unit
		if unit == "" {
			unit = "dona"
		}
		// Line 1: Item name
		buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Name))
		// Line 2: Qty x Price = Total
		qtyStr := fmt.Sprintf("%.3f", item.Quantity)
		qtyStr = strings.TrimRight(strings.TrimRight(qtyStr, "0"), ".")
		line2 := fmt.Sprintf("   %s %s x %s = %s so'm\n", qtyStr, unit, FormatMoney(item.UnitPrice), FormatMoney(item.TotalPrice))
		buf.WriteString(line2)
	}
	buf.WriteString("--------------------------------\n")

	// 6. Total Amount (Right align, Bold, Double Height)
	buf.Write([]byte("\x1b\x61\x02")) // Right align
	buf.Write([]byte("\x1b\x45\x01")) // Bold ON
	buf.Write([]byte("\x1d\x21\x01")) // Double height
	buf.WriteString(fmt.Sprintf("JAMI: %s SO'M\n", FormatMoney(req.TotalAmount)))
	buf.Write([]byte("\x1d\x21\x00")) // Normal size
	buf.Write([]byte("\x1b\x45\x00")) // Bold OFF

	// 7. Cash given & change (if any)
	if req.CashGiven > 0 && req.PaymentMethod == "cash" {
		buf.WriteString(fmt.Sprintf("To'langan: %s so'm\n", FormatMoney(req.CashGiven)))
		if req.Change >= 0 {
			buf.WriteString(fmt.Sprintf("Qaytim: %s so'm\n", FormatMoney(req.Change)))
		}
	}
	buf.WriteString("================================\n")

	// 8. Footer (Center)
	buf.Write([]byte("\x1b\x61\x01")) // Center
	buf.WriteString("Xaridingiz uchun rahmat!\n")
	buf.WriteString("Yana tashrif buyuring!\n")
	buf.WriteString("ORZU POS - Oflayn tizim\n")

	// 9. Feed lines & Cut Paper
	buf.WriteString("\n\n\n\n")
	buf.Write([]byte("\x1d\x56\x41\x03")) // GS V 65 3: Feed 3 lines and full cut

	return buf.Bytes()
}

// BuildEscPosSingleLabel constructs a single barcode sticker label
func BuildEscPosSingleLabel(req LabelRequest) []byte {
	var buf bytes.Buffer

	// 1. Initialize printer
	buf.Write([]byte("\x1b\x40"))
	buf.Write([]byte("\x1b\x61\x01")) // Center alignment

	// 2. Store Name
	if req.ShowStore {
		storeName := strings.TrimSpace(req.StoreName)
		if storeName == "" {
			storeName = "OLMOS ORZU MARKET"
		}
		buf.Write([]byte("\x1b\x45\x01")) // Bold ON
		buf.WriteString(strings.ToUpper(storeName) + "\n")
		buf.Write([]byte("\x1b\x45\x00")) // Bold OFF
	}

	// 3. Product Name (Bold)
	buf.Write([]byte("\x1b\x45\x01")) // Bold ON
	buf.WriteString(req.Name + "\n")
	buf.Write([]byte("\x1b\x45\x00")) // Bold OFF

	// 4. Short Code
	if req.ShortCode != "" {
		buf.WriteString(fmt.Sprintf("KOD: %s\n", req.ShortCode))
	}

	// 5. Barcode (Code128 or EAN13)
	code := strings.TrimSpace(req.Barcode)
	if code != "" {
		buf.Write([]byte("\x1d\x68\x40")) // Barcode height (64 dots)
		buf.Write([]byte("\x1d\x77\x02")) // Barcode width (2)
		buf.Write([]byte("\x1d\x48\x02")) // HRI font below barcode
		buf.Write([]byte("\x1d\x66\x00")) // Font A

		isDigits := true
		for _, r := range code {
			if r < '0' || r > '9' {
				isDigits = false
				break
			}
		}

		if len(code) == 13 && isDigits {
			// EAN-13
			buf.Write([]byte("\x1d\x6b\x02"))
			buf.WriteString(code)
			buf.WriteByte(0)
		} else if len(code) == 8 && isDigits {
			// EAN-8
			buf.Write([]byte("\x1d\x6b\x01"))
			buf.WriteString(code)
			buf.WriteByte(0)
		} else {
			// Code 128 (Subset B)
			payload := append([]byte{0x7b, 0x42}, []byte(code)...)
			buf.Write([]byte{0x1d, 0x6b, 0x49, byte(len(payload))})
			buf.Write(payload)
		}
		buf.WriteString("\n")
	}

	// 6. Price (Double Height & Width)
	if req.ShowPrice && req.Price > 0 {
		unit := req.Unit
		if unit == "" {
			unit = "dona"
		}
		buf.Write([]byte("\x1b\x45\x01")) // Bold ON
		buf.Write([]byte("\x1d\x21\x11")) // Double Width & Height
		buf.WriteString(fmt.Sprintf("%s so'm\n", FormatMoney(req.Price)))
		buf.Write([]byte("\x1d\x21\x00")) // Normal size
		buf.Write([]byte("\x1b\x45\x00")) // Bold OFF
		buf.WriteString(fmt.Sprintf("1 %s uchun\n", unit))
	}

	// Separator
	if req.LabelWidth == "40x30" || req.LabelWidth == "40" {
		buf.WriteString("----------------\n")
	} else if req.LabelWidth == "58" {
		buf.WriteString("----------------------\n")
	} else {
		buf.WriteString("--------------------------------\n")
	}

	// Feed lines
	buf.WriteString("\n\n")

	// Cut after label if requested
	if req.AutoCut {
		buf.Write([]byte("\x1d\x56\x41\x02")) // GS V 65 2: Feed 2 lines & cut
	}

	return buf.Bytes()
}

// PrintReceipt prints directly to default Windows printer
func PrintReceipt(req ReceiptRequest) (string, error) {
	printerName, err := GetDefaultPrinter()
	if err != nil {
		return "", fmt.Errorf("standart printer topilmadi: %v", err)
	}

	docName := "ORZU POS Chek - " + req.ReceiptNo
	rawBytes := BuildEscPosReceipt(req)

	err = PrintRawBytes(printerName, docName, rawBytes)
	if err != nil {
		return printerName, err
	}
	return printerName, nil
}

// BuildEscPos40x30Label constructs an ultra-compact binary ESC/POS label for 40x30 mm stickers
func BuildEscPos40x30Label(req LabelRequest) []byte {
	var buf bytes.Buffer

	// 1. Initialize & Center & Compact line spacing (18 dots)
	buf.Write([]byte("\x1b\x40"))     // Initialize printer
	buf.Write([]byte("\x1b\x61\x01")) // Center alignment
	buf.Write([]byte("\x1b\x33\x12")) // Line spacing 18 dots (compact)

	// 2. Store Name (Font B, Bold, max 18 chars to fit 40mm)
	if req.ShowStore {
		storeName := strings.TrimSpace(req.StoreName)
		if storeName == "" {
			storeName = "OLMOS ORZU MARKET"
		}
		if len(storeName) > 18 {
			storeName = storeName[:18]
		}
		buf.Write([]byte("\x1b\x4d\x01")) // Font B (9x17 dots, fits 40mm)
		buf.Write([]byte("\x1b\x45\x01")) // Bold ON
		buf.WriteString(strings.ToUpper(storeName) + "\n")
		buf.Write([]byte("\x1b\x45\x00")) // Bold OFF
		buf.Write([]byte("\x1b\x4d\x00")) // Font A
	}

	// 3. Product Name (Font A, Bold, max 16 chars to avoid horizontal overflow)
	name := strings.TrimSpace(req.Name)
	if len(name) > 16 {
		name = name[:16]
	}
	buf.Write([]byte("\x1b\x45\x01")) // Bold ON
	buf.WriteString(name + "\n")
	buf.Write([]byte("\x1b\x45\x00")) // Bold OFF

	// 4. Barcode (Height: 36 dots, Width: 2, HRI Font B below)
	code := strings.TrimSpace(req.Barcode)
	if code != "" {
		buf.Write([]byte("\x1d\x68\x24")) // Barcode height 36 dots
		buf.Write([]byte("\x1d\x77\x02")) // Barcode width 2 dots
		buf.Write([]byte("\x1d\x48\x02")) // HRI font below barcode
		buf.Write([]byte("\x1d\x66\x01")) // HRI Font B (compact)

		isDigits := true
		for _, r := range code {
			if r < '0' || r > '9' {
				isDigits = false
				break
			}
		}

		if len(code) == 13 && isDigits {
			// EAN-13
			buf.Write([]byte("\x1d\x6b\x02"))
			buf.WriteString(code)
			buf.WriteByte(0)
		} else if len(code) == 8 && isDigits {
			// EAN-8
			buf.Write([]byte("\x1d\x6b\x01"))
			buf.WriteString(code)
			buf.WriteByte(0)
		} else {
			// Code 128 (Subset B)
			payload := append([]byte{0x7b, 0x42}, []byte(code)...)
			buf.Write([]byte{0x1d, 0x6b, 0x49, byte(len(payload))})
			buf.Write(payload)
		}
		buf.WriteString("\n")
	}

	// 5. Price (Double Height ONLY, Single Width, Bold)
	if req.ShowPrice && req.Price > 0 {
		buf.Write([]byte("\x1b\x45\x01")) // Bold ON
		buf.Write([]byte("\x1d\x21\x01")) // Double Height ONLY (Normal width = Never overflows 40mm)
		buf.WriteString(fmt.Sprintf("%s so'm\n", FormatMoney(req.Price)))
		buf.Write([]byte("\x1d\x21\x00")) // Normal size
		buf.Write([]byte("\x1b\x45\x00")) // Bold OFF
	}

	// 6. Reset line spacing & 1 single clean line feed (NO cut, NO extra empty space)
	buf.Write([]byte("\x1b\x32")) // Default line spacing
	buf.WriteString("\n")

	return buf.Bytes()
}

// PrintLabel prints barcode sticker labels with requested copies
func PrintLabel(req LabelRequest) (string, error) {
	printerName, err := GetDefaultPrinter()
	if err != nil {
		return "", fmt.Errorf("standart printer topilmadi: %v", err)
	}

	copies := req.Copies
	if copies <= 0 {
		copies = 1
	}
	if copies > 100 {
		copies = 100
	}

	var batch bytes.Buffer
	for i := 0; i < copies; i++ {
		if req.LabelWidth == "40x30" || req.LabelWidth == "40" {
			batch.Write(BuildEscPos40x30Label(req))
		} else {
			batch.Write(BuildEscPosSingleLabel(req))
		}
	}

	// If individual autocut was requested and not 40mm
	if req.AutoCut && req.LabelWidth != "40x30" && req.LabelWidth != "40" {
		batch.WriteString("\n\n")
		batch.Write([]byte("\x1d\x56\x41\x02"))
	}

	docName := fmt.Sprintf("ORZU POS Yorliq - %s (%d nusxa)", req.Name, copies)
	err = PrintRawBytes(printerName, docName, batch.Bytes())
	if err != nil {
		return printerName, err
	}
	return printerName, nil
}
