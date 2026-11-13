# ORZU POS — Olmos Orzu Market

**Namangan davlat pedagogika instituti**  
**Raqamli taʼlim texnologiyalari markazi (NamDPI RTTM)** tomonidan ishlab chiqilgan professional chakana savdo va supermarket avtomatlashtirish tizimi (Point of Sale).

---

## 🏗 Ekotizim Tuzilishi (Tayyor POS kabi 5 ta modul):

| Modul | Dastur | Vazifasi |
| :--- | :--- | :--- |
| **1-qadam** | **ORZU DB** (`orzu-db.exe`) | PostgreSQL / SQLite avtomatik sozlovchi va jadvallar migratsiyasi |
| **2-qadam** | **ORZU Server** (`orzu-server.exe`) | Go Core REST API, Real-time WebSocket, markaziy maʼlumotlar bazasi |
| **3-qadam** | **ORZU Boshqaruv** (`/admin`) | Tovarlar ombori, kirim, narxlar, hisobotlar, tahlil va xodimlar |
| **4-qadam** | **ORZU Kassa** (`orzu-kassa.exe`) | Kassir sotuv oynasi (Shtrix-kod, sensorli ekran, tezkor toʻlov, chek printer) |
| **5-qadam** | **ORZU Arxiv** (`orzu-backup.exe`) | Avtomatik va 1-klikli zaxira nusxa (Backup & Restore) vositasi |

---

## ⚡ Asosiy Imkoniyatlar:
1. **Tezkor Kassa (Sotuv):**
   - Shtrix-kod skanerini toʻgʻridan-toʻgʻri `Enter` bilan qabul qilish.
   - Tezkor klaviatura tugmalari: `F2` (Naqd pul), `F3` (Karta), `F4` (Click/Payme), `F5` (Nasiya), `Esc` (Bekor qilish).
   - Qaytimni avtomatik hisoblash va tezkor kupyuralar (10k, 20k, 50k, 100k, 200k).
2. **Nasiya daftari (Qarzdorlik):**
   - Doimiy mijozlarga qarzga tovar berish va qarz toʻlovlarini qabul qilish.
3. **Lokal Tarmoq (LAN Multi-till):**
   - Bitta server kompyuterga doʻkondagi barcha kassa kompyuterlari, telefon va planshetlar simsiz (Wi-Fi) ulanadi.
4. **Oflayn va Ishonchli:**
   - Internet uzilib qolsa ham SQLite / PostgreSQL mahalliy rejimda toʻxtovsiz ishlaydi.
   - Xotira sarfi: atigi **20–30 MB RAM** (Electronʼdan 20 barobar yengil).
5. **Termal Chek chiqarish (ESC/POS):**
   - 58mm va 80mm chek printerlariga toʻliq moslashtirilgan.

---

## 🚀 Ishga tushirish (Kompilyatsiya):

```bash
# 1. Paketlarni yuklash
go mod tidy

# 2. Serverni ishga tushirish
go run cmd/orzu-server/main.go

# 3. Kassa oynasini ochish
go run cmd/orzu-kassa/main.go
```

Kassa ilovasi: **`http://localhost:8080`**  
Boshqaruv paneli: **`http://localhost:8080/admin`**
