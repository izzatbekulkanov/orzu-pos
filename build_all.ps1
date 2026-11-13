$env:GOPATH = "D:\go_workspace"
$env:GOCACHE = "D:\go_workspace\cache"
$env:GOMODCACHE = "D:\go_workspace\pkg\mod"
$goExe = "D:\Go\bin\go.exe"

Set-Location "D:\Github\orzu-pos"

Write-Host "Building ORZU POS ecosystem (GUI Mode - No Console Windows)..."

# 1. Server (with System Tray)
Write-Host "1/5 Building orzu-server.exe (with System Tray)..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/orzu-server.exe ./cmd/orzu-server

# 2. Kassa
Write-Host "2/5 Building orzu-kassa.exe (Clean Desktop App)..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/orzu-kassa.exe ./cmd/orzu-kassa

# 3. Setup / Installer Hub
Write-Host "3/5 Building ORZU_Setup.exe..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/ORZU_Setup.exe ./cmd/orzu-installer

# 4. DB
Write-Host "4/5 Building orzu-db.exe..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/orzu-db.exe ./cmd/orzu-db

# 5. Backup / Arxiv
Write-Host "5/5 Building orzu-backup.exe..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/orzu-backup.exe ./cmd/orzu-backup

Write-Host "Build complete! Updating D:\ORZU_POS if present..."
if (Test-Path "D:\ORZU_POS") {
    Copy-Item -Path "bin\*" -Destination "D:\ORZU_POS\" -Force
    Write-Host "Updated D:\ORZU_POS binaries successfully."
}

Write-Host "All binaries are now compiled in pure GUI mode with NO terminal windows!"
