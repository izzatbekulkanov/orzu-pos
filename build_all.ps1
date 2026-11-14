$env:GOPATH = "D:\go_workspace"
$env:GOCACHE = "D:\go_workspace\cache"
$env:GOMODCACHE = "D:\go_workspace\pkg\mod"
$goExe = "D:\Go\bin\go.exe"

Set-Location "D:\Github\orzu-pos"

Write-Host "Stopping running ORZU POS processes if any..."
Get-Process -Name "ORZU_Setup", "orzu-server", "orzu-kassa", "orzu-admin", "orzu-db", "orzu-backup" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 600

Write-Host "Building ORZU POS ecosystem with EMBEDDED ICONS..."

# 1. Server (Console Terminal with live logs & System Tray)
Write-Host "1/6 Building orzu-server.exe..."
& $goExe build -ldflags "-s -w" -o bin/orzu-server.exe ./cmd/orzu-server

# 2. Kassa (Clean Desktop App & Icon)
Write-Host "2/6 Building orzu-kassa.exe..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/orzu-kassa.exe ./cmd/orzu-kassa

# 3. Admin Panel (Clean Desktop App & Icon)
Write-Host "3/6 Building orzu-admin.exe..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/orzu-admin.exe ./cmd/orzu-admin

# 4. Setup / Installer Hub
Write-Host "4/6 Building ORZU_Setup.exe..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/ORZU_Setup.exe ./cmd/orzu-installer

# 5. DB
Write-Host "5/6 Building orzu-db.exe..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/orzu-db.exe ./cmd/orzu-db

# 6. Backup / Arxiv
Write-Host "6/6 Building orzu-backup.exe..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/orzu-backup.exe ./cmd/orzu-backup

Write-Host "Copying to D:\ORZU_POS..."
New-Item -ItemType Directory -Force -Path "D:\ORZU_POS\assets\icons" | Out-Null
New-Item -ItemType Directory -Force -Path "D:\ORZU_POS\uploads\products" | Out-Null
Copy-Item -Path "bin\*" -Destination "D:\ORZU_POS\" -Force
Copy-Item -Path "assets\icons\*" -Destination "D:\ORZU_POS\assets\icons\" -Force
Copy-Item -Path "web" -Destination "D:\ORZU_POS\" -Recurse -Force
Copy-Item -Path "version.json" -Destination "D:\ORZU_POS\" -Force -ErrorAction SilentlyContinue

# Clean up old Desktop shortcuts and create fresh ones
$desktopDir = [Environment]::GetFolderPath('Desktop')
Write-Host "Cleaning up old desktop shortcuts..."
Get-ChildItem -Path $desktopDir -Filter "*ORZU*.lnk" -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue
Get-ChildItem -Path $desktopDir -Filter "*orzu*.lnk" -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue

Write-Host "Generating fresh desktop shortcuts with embedded icons..."
$wsh = New-Object -ComObject WScript.Shell

# 1. ORZU Kassa shortcut
$sKassa = $wsh.CreateShortcut((Join-Path $desktopDir "ORZU Kassa.lnk"))
$sKassa.TargetPath = "D:\ORZU_POS\orzu-kassa.exe"
$sKassa.WorkingDirectory = "D:\ORZU_POS"
$sKassa.IconLocation = "D:\ORZU_POS\assets\icons\kassa.ico"
$sKassa.Save()

# 2. ORZU Boshqaruv shortcut
$sAdmin = $wsh.CreateShortcut((Join-Path $desktopDir "ORZU Boshqaruv.lnk"))
$sAdmin.TargetPath = "D:\ORZU_POS\orzu-admin.exe"
$sAdmin.WorkingDirectory = "D:\ORZU_POS"
$sAdmin.IconLocation = "D:\ORZU_POS\assets\icons\admin.ico"
$sAdmin.Save()

# 3. ORZU Setup shortcut
$sSetup = $wsh.CreateShortcut((Join-Path $desktopDir "ORZU O'rnatuvchi.lnk"))
$sSetup.TargetPath = "D:\ORZU_POS\ORZU_Setup.exe"
$sSetup.WorkingDirectory = "D:\ORZU_POS"
$sSetup.IconLocation = "D:\ORZU_POS\assets\icons\installer.ico"
$sSetup.Save()

# 4. ORZU DB shortcut
$sDB = $wsh.CreateShortcut((Join-Path $desktopDir "ORZU DB.lnk"))
$sDB.TargetPath = "D:\ORZU_POS\orzu-db.exe"
$sDB.WorkingDirectory = "D:\ORZU_POS"
$sDB.IconLocation = "D:\ORZU_POS\assets\icons\db.ico"
$sDB.Save()

# 5. ORZU Arxiv shortcut
$sBackup = $wsh.CreateShortcut((Join-Path $desktopDir "ORZU Arxiv.lnk"))
$sBackup.TargetPath = "D:\ORZU_POS\orzu-backup.exe"
$sBackup.WorkingDirectory = "D:\ORZU_POS"
$sBackup.IconLocation = "D:\ORZU_POS\assets\icons\backup.ico"
$sBackup.Save()

# 6. ORZU Server shortcut
$sServer = $wsh.CreateShortcut((Join-Path $desktopDir "ORZU Server.lnk"))
$sServer.TargetPath = "D:\ORZU_POS\orzu-server.exe"
$sServer.WorkingDirectory = "D:\ORZU_POS"
$sServer.IconLocation = "D:\ORZU_POS\assets\icons\server.ico"
$sServer.Save()

Write-Host "All 6 binaries compiled, copied to D:\ORZU_POS, and fresh Desktop shortcuts created successfully!"
