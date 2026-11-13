package tray

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procLoadIconW           = user32.NewProc("LoadIconW")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")

	procShell_NotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

const (
	WM_USER         = 0x0400
	WM_TRAYCALLBACK = WM_USER + 1
	WM_RBUTTONUP    = 0x0205
	WM_LBUTTONUP    = 0x0202
	WM_COMMAND      = 0x0111
	WM_DESTROY      = 0x0002

	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_INFO    = 0x00000010

	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800

	TPM_BOTTOMALIGN = 0x0020
	TPM_LEFTALIGN   = 0x0000

	ID_MENU_KASSA    = 1001
	ID_MENU_ADMIN    = 1002
	ID_MENU_INSTALLER= 1003
	ID_MENU_EXIT     = 1004
)

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type POINT struct {
	X int32
	Y int32
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	TimeoutOrVersion uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

var (
	globalHWnd uintptr
	globalNID  NOTIFYICONDATA
	serverPort = "8080"
)

func wndProc(hWnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYCALLBACK:
		switch lParam {
		case WM_RBUTTONUP, WM_LBUTTONUP:
			showMenu(hWnd)
		}
	case WM_COMMAND:
		menuID := int(wParam & 0xFFFF)
		switch menuID {
		case ID_MENU_KASSA:
			OpenApp("http://localhost:" + serverPort)
		case ID_MENU_ADMIN:
			OpenApp("http://localhost:" + serverPort + "/admin")
		case ID_MENU_INSTALLER:
			OpenApp("http://localhost:8899")
		case ID_MENU_EXIT:
			procShell_NotifyIconW.Call(uintptr(NIM_DELETE), uintptr(unsafe.Pointer(&globalNID)))
			procPostQuitMessage.Call(0)
			os.Exit(0)
		}
	case WM_DESTROY:
		procShell_NotifyIconW.Call(uintptr(NIM_DELETE), uintptr(unsafe.Pointer(&globalNID)))
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hWnd, uintptr(msg), wParam, lParam)
	return r
}

func showMenu(hWnd uintptr) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}

	appendMenuItem(hMenu, ID_MENU_KASSA, "🛒 ORZU Kassa (Sotuv ekrani)")
	appendMenuItem(hMenu, ID_MENU_ADMIN, "📊 ORZU Boshqaruv (Admin panel)")
	appendMenuItem(hMenu, ID_MENU_INSTALLER, "📦 O'rnatuvchi markazi (Setup)")
	appendSeparator(hMenu)
	appendMenuItem(hMenu, ID_MENU_EXIT, "🚪 Chiqish (Serverni to'xtatish)")

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	procSetForegroundWindow.Call(hWnd)
	procTrackPopupMenu.Call(
		hMenu,
		uintptr(TPM_BOTTOMALIGN|TPM_LEFTALIGN),
		uintptr(pt.X),
		uintptr(pt.Y),
		0,
		hWnd,
		0,
	)
}

func appendMenuItem(hMenu uintptr, id uintptr, title string) {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), id, uintptr(unsafe.Pointer(titlePtr)))
}

func appendSeparator(hMenu uintptr) {
	procAppendMenuW.Call(hMenu, uintptr(MF_SEPARATOR), 0, 0)
}

func InitTray(port string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	serverPort = port

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className, _ := syscall.UTF16PtrFromString("OrzuPosTrayClass")

	var wc WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(wndProc)
	wc.HInstance = hInstance
	wc.LpszClassName = className

	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	hWnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(className)),
		0,
		0, 0, 0, 0,
		0, 0,
		hInstance,
		0,
	)
	if hWnd == 0 {
		return fmt.Errorf("failed to create tray window")
	}
	globalHWnd = hWnd

	// Load application icon embedded via rsrc (ID = 1), or fallback to standard IDI_APPLICATION
	hIcon, _, _ := procLoadIconW.Call(hInstance, uintptr(1))
	if hIcon == 0 {
		hIcon, _, _ = procLoadIconW.Call(0, uintptr(32512))
	}

	globalNID.CbSize = uint32(unsafe.Sizeof(globalNID))
	globalNID.HWnd = hWnd
	globalNID.UID = 1
	globalNID.UFlags = NIF_ICON | NIF_MESSAGE | NIF_TIP | NIF_INFO
	globalNID.UCallbackMessage = WM_TRAYCALLBACK
	globalNID.HIcon = hIcon

	tip, _ := syscall.UTF16FromString(fmt.Sprintf("ORZU POS Server (:%s) — Faol", port))
	copy(globalNID.SzTip[:], tip)

	// Balloon notification
	balloonTitle, _ := syscall.UTF16FromString("ORZU POS Ishga tushdi")
	copy(globalNID.SzInfoTitle[:], balloonTitle)
	balloonInfo, _ := syscall.UTF16FromString(fmt.Sprintf("Server :%s portida faol. Kassa uchun bosing.", port))
	copy(globalNID.SzInfo[:], balloonInfo)
	globalNID.DwInfoFlags = 0x00000001 // NIIF_INFO

	procShell_NotifyIconW.Call(uintptr(NIM_ADD), uintptr(unsafe.Pointer(&globalNID)))

	// Run message pump in this thread (blocks)
	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	return nil
}

// OpenApp opens the given URL in Edge app mode or browser without showing any console window
func OpenApp(url string) {
	edgePath := `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`
	if _, err := os.Stat(edgePath); err == nil {
		cmd := exec.Command(edgePath, "--app="+url, "--force-dark-mode", "--start-maximized")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Start()
		return
	}

	edge64 := `C:\Program Files\Microsoft\Edge\Application\msedge.exe`
	if _, err := os.Stat(edge64); err == nil {
		cmd := exec.Command(edge64, "--app="+url, "--force-dark-mode", "--start-maximized")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Start()
		return
	}

	// Fallback to rundll32 url.dll,FileProtocolHandler (opens default browser without CMD window)
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
}
