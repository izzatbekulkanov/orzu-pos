package license

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type LicenseInfo struct {
	InstalledAt  time.Time  `json:"installed_at"`
	IsActivated  bool       `json:"is_activated"`
	LicenseKey   string     `json:"license_key,omitempty"`
	ActivatedAt  *time.Time `json:"activated_at,omitempty"`
	CustomerName string     `json:"customer_name"`
}

type StatusResponse struct {
	IsActivated  bool   `json:"is_activated"`
	IsTrial      bool   `json:"is_trial"`
	DaysLeft     int    `json:"days_left"`
	IsExpired    bool   `json:"is_expired"`
	CustomerName string `json:"customer_name"`
	InstalledAt  string `json:"installed_at"`
	LicenseKey   string `json:"license_key,omitempty"`
}

var (
	licenseMu sync.Mutex
)

func getLicenseFilePath() string {
	candidates := []string{
		`D:\ORZU_POS\data\license.json`,
		filepath.Join("data", "license.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if _, err := os.Stat(`D:\ORZU_POS`); err == nil {
		_ = os.MkdirAll(`D:\ORZU_POS\data`, 0755)
		return `D:\ORZU_POS\data\license.json`
	}
	_ = os.MkdirAll("data", 0755)
	return filepath.Join("data", "license.json")
}

func LoadLicense() LicenseInfo {
	licenseMu.Lock()
	defer licenseMu.Unlock()

	filePath := getLicenseFilePath()
	var info LicenseInfo

	data, err := os.ReadFile(filePath)
	if err == nil {
		if json.Unmarshal(data, &info) == nil && !info.InstalledAt.IsZero() {
			return info
		}
	}

	// First run: Initialize 30-day trial
	info = LicenseInfo{
		InstalledAt:  time.Now(),
		IsActivated:  false,
		CustomerName: "Olmos Orzu market",
	}

	saveLicenseUnsafe(info, filePath)
	return info
}

func saveLicenseUnsafe(info LicenseInfo, path string) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	bytes, err := json.MarshalIndent(info, "", "  ")
	if err == nil {
		_ = os.WriteFile(path, bytes, 0644)
	}
	// Also mirror to D:\ORZU_POS if different
	mirrorPath := `D:\ORZU_POS\data\license.json`
	if path != mirrorPath && isDirExists(`D:\ORZU_POS`) {
		_ = os.MkdirAll(`D:\ORZU_POS\data`, 0755)
		_ = os.WriteFile(mirrorPath, bytes, 0644)
	}
}

func isDirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func GetStatus() StatusResponse {
	info := LoadLicense()

	resp := StatusResponse{
		IsActivated:  info.IsActivated,
		CustomerName: info.CustomerName,
		InstalledAt:  info.InstalledAt.Format("02.01.2006"),
		LicenseKey:   info.LicenseKey,
	}

	if info.IsActivated {
		resp.IsTrial = false
		resp.DaysLeft = 9999
		resp.IsExpired = false
		return resp
	}

	elapsedDays := int(time.Since(info.InstalledAt).Hours() / 24)
	daysLeft := 30 - elapsedDays

	if daysLeft <= 0 {
		resp.IsTrial = false
		resp.DaysLeft = 0
		resp.IsExpired = true
	} else {
		resp.IsTrial = true
		resp.DaysLeft = daysLeft
		resp.IsExpired = false
	}

	return resp
}

func ValidateAndActivate(key string) (bool, string) {
	key = strings.ToUpper(strings.TrimSpace(key))
	if key == "" {
		return false, "Aktivatsiya kalitini kiriting"
	}

	// Master commercial keys
	validKeys := map[string]bool{
		"ORZU-OLMOS-2026-MARKET-7777": true,
		"ORZU-NMDPI-2026-RTTM-8888":   true,
		"ORZU-PRO-FULL-ACCESS-2026":   true,
	}

	isValid := validKeys[key]
	if !isValid {
		// Algorithmic validation: ORZU-XXXX-YYYY-ZZZZ-HASH
		parts := strings.Split(key, "-")
		if len(parts) == 5 && parts[0] == "ORZU" {
			h := sha256.Sum256([]byte(parts[1] + parts[2] + parts[3] + "ORZU_SALT"))
			checksum := strings.ToUpper(hex.EncodeToString(h[:]))[:4]
			if parts[4] == checksum {
				isValid = true
			}
		}
	}

	if !isValid {
		return false, "Kiritilgan aktivatsiya kaliti noto'g'ri yoki yaroqsiz"
	}

	licenseMu.Lock()
	defer licenseMu.Unlock()

	filePath := getLicenseFilePath()
	var info LicenseInfo
	data, err := os.ReadFile(filePath)
	if err == nil {
		_ = json.Unmarshal(data, &info)
	}
	if info.InstalledAt.IsZero() {
		info.InstalledAt = time.Now()
	}

	now := time.Now()
	info.IsActivated = true
	info.LicenseKey = key
	info.ActivatedAt = &now
	if info.CustomerName == "" {
		info.CustomerName = "Olmos Orzu market"
	}

	saveLicenseUnsafe(info, filePath)
	return true, fmt.Sprintf("ORZU POS muvaffaqiyatli faollashtirildi! Litsenziat: %s", info.CustomerName)
}
