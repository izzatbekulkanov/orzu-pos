$env:GOPATH = "D:\go_workspace"
$env:GOCACHE = "D:\go_workspace\cache"
$env:GOMODCACHE = "D:\go_workspace\pkg\mod"
$goExe = "D:\Go\bin\go.exe"

Set-Location "D:\Github\orzu-pos"

Write-Host "Building ORZU POS ecosystem with EMBEDDED ICONS..."

# 1. Server (with System Tray & Icon)
Write-Host "1/5 Building orzu-server.exe..."
& $goExe build -ldflags "-H=windowsgui -s -w" -o bin/orzu-server.exe ./cmd/orzu-server

# 2. Kassa (Clean Desktop App & Icon)
Write-Host "2/5 Building orzu-kassa.exe..."
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

Write-Host "Copying to D:\ORZU_POS..."
New-Item -ItemType Directory -Force -Path "D:\ORZU_POS\assets\icons" | Out-Null
Copy-Item -Path "bin\*" -Destination "D:\ORZU_POS\" -Force
Copy-Item -Path "assets\icons\*" -Destination "D:\ORZU_POS\assets\icons\" -Force

# Create Desktop Shortcuts
$desktopDir = [Environment]::GetFolderPath('Desktop')
$wsh = New-Object -ComObject WScript.Shell

# 1. ORZU Kassa shortcut
$sKassa = $wsh.CreateShortcut((Join-Path $desktopDir "ORZU Kassa.lnk"))
$sKassa.TargetPath = "D:\ORZU_POS\orzu-kassa.exe"
$sKassa.WorkingDirectory = "D:\ORZU_POS"
$sKassa.IconLocation = "D:\ORZU_POS\assets\icons\kassa.ico"
$sKassa.Save()

# 2. ORZU Boshqaruv shortcut
$sAdmin = $wsh.CreateShortcut((Join-Path $desktopDir "ORZU Boshqaruv.lnk"))
$sAdmin.TargetPath = "C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
$sAdmin.Arguments = "--app=http://localhost:8080/admin --force-dark-mode --start-maximized"
$sAdmin.IconLocation = "D:\ORZU_POS\assets\icons\admin.ico"
$sAdmin.Save()

# 3. ORZU Setup shortcut
$sSetup = $wsh.CreateShortcut((Join-Path $desktopDir "ORZU O'rnatuvchi.lnk"))
$sSetup.TargetPath = "D:\ORZU_POS\ORZU_Setup.exe"
$sSetup.WorkingDirectory = "D:\ORZU_POS"
$sSetup.IconLocation = "D:\ORZU_POS\assets\icons\installer.ico"
$sSetup.Save()

Write-Host "All binaries compiled with embedded icons, copied to D:\ORZU_POS, and Desktop shortcuts created!"
