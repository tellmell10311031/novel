//go:build windows

package main

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procGetDpiForSystem  = user32.NewProc("GetDpiForSystem")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procGetWindowLongPtr = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr = user32.NewProc("SetWindowLongPtrW")
	procGetWindowRect    = user32.NewProc("GetWindowRect")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procMonitorFromWnd   = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo   = user32.NewProc("GetMonitorInfoW")
)

const (
	wsOverlappedWindow = 0x00CF0000
	swpNoZOrder        = 0x0004
	swpNoOwnerZOrder   = 0x0200
	swpFrameChanged    = 0x0020
	monitorNearest     = 2
	webView2URL        = "https://go.microsoft.com/fwlink/p/?LinkId=2124703"
)

type rect struct{ Left, Top, Right, Bottom int32 }
type monitorInfo struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
}

func messageBox(title, text string) {
	t, _ := windows.UTF16PtrFromString(text)
	c, _ := windows.UTF16PtrFromString(title)
	windows.MessageBox(0, t, c, 0x10) // MB_ICONERROR
}

func systemDPI() int {
	if procGetDpiForSystem.Find() == nil {
		if d, _, _ := procGetDpiForSystem.Call(); d > 0 {
			return int(d)
		}
	}
	return 96
}

// 画面の DPI と大きさに合わせてウィンドウサイズを決める（縦横比は維持）
func windowSize(w, h int) (uint, uint) {
	dpi := systemDPI()
	w, h = w*dpi/96, h*dpi/96
	sw, _, _ := procGetSystemMetrics.Call(0) // SM_CXSCREEN
	sh, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
	maxW, maxH := int(sw)*9/10, int(sh)*85/100
	if maxW > 0 && w > maxW {
		h, w = h*maxW/w, maxW
	}
	if maxH > 0 && h > maxH {
		w, h = w*maxH/h, maxH
	}
	return uint(w), uint(h)
}

type fullscreen struct {
	on    bool
	style uintptr
	r     rect
}

func (f *fullscreen) toggle(hwnd uintptr) bool {
	gwlStyle := -16
	if !f.on {
		f.style, _, _ = procGetWindowLongPtr.Call(hwnd, uintptr(gwlStyle))
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&f.r)))
		mon, _, _ := procMonitorFromWnd.Call(hwnd, monitorNearest)
		mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
		procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi)))
		procSetWindowLongPtr.Call(hwnd, uintptr(gwlStyle), f.style&^wsOverlappedWindow)
		m := mi.RcMonitor
		procSetWindowPos.Call(hwnd, 0, uintptr(m.Left), uintptr(m.Top),
			uintptr(m.Right-m.Left), uintptr(m.Bottom-m.Top), swpNoOwnerZOrder|swpFrameChanged)
		f.on = true
	} else {
		procSetWindowLongPtr.Call(hwnd, uintptr(gwlStyle), f.style)
		procSetWindowPos.Call(hwnd, 0, uintptr(f.r.Left), uintptr(f.r.Top),
			uintptr(f.r.Right-f.r.Left), uintptr(f.r.Bottom-f.r.Top), swpNoZOrder|swpNoOwnerZOrder|swpFrameChanged)
		f.on = false
	}
	return f.on
}

func main() {
	runtime.LockOSThread()

	exe, err := os.Executable()
	if err != nil {
		messageBox("エラー", err.Error())
		return
	}
	p, err := readPayloadFile(exe)
	if err != nil {
		messageBox("エラー", "ゲームデータを読み込めませんでした。\n"+
			"このファイルはノベルゲームメーカーの「EXE出力」で作られたものではないか、壊れています。\n\n"+err.Error())
		return
	}

	appData, err := os.UserConfigDir() // %APPDATA%
	if err != nil {
		appData = filepath.Dir(exe)
	}
	saves := newStore(appData, p.Title, p.ID)
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = appData
	}
	dataPath := filepath.Join(localAppData, "NovelGames", folderName(p.Title, p.ID), "WebView2")

	// ゲーム本体はローカル（127.0.0.1）だけで配信する
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		messageBox("エラー", "起動に失敗しました。\n"+err.Error())
		return
	}
	html := []byte(p.HTML)
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(html)
	}))

	width, height := windowSize(p.Width, p.Height)
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		DataPath:  dataPath,
		WindowOptions: webview2.WindowOptions{
			Title:  p.Title,
			Width:  width,
			Height: height,
			IconId: 1,
			Center: true,
		},
	})
	if w == nil {
		messageBox(p.Title, "このゲームの起動には Microsoft Edge WebView2 ランタイムが必要です。\n"+
			"（Windows 10 / 11 には通常インストールされています）\n\n"+
			"次のページからインストールしてから、もう一度起動してください。\n"+webView2URL)
		return
	}
	defer w.Destroy()

	hwnd := uintptr(w.Window())
	fs := &fullscreen{}
	w.Bind("novelLoad", func(key string) (string, error) { return saves.Load(key) })
	w.Bind("novelSave", func(key, value string) error { return saves.Save(key, value) })
	w.Bind("novelDelete", func(key string) error { return saves.Delete(key) })
	w.Bind("novelFullscreen", func() bool { return fs.toggle(hwnd) })
	w.Bind("novelQuit", func() { w.Dispatch(w.Terminate) })
	w.Init("window.__NOVEL_EXE__ = true;")
	w.Navigate("http://" + ln.Addr().String() + "/")
	w.Run()
}
