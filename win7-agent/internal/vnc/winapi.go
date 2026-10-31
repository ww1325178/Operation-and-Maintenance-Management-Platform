package vnc

import "syscall"

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")

	// user32
	procGetDC            = user32.NewProc("GetDC")
	procReleaseDC        = user32.NewProc("ReleaseDC")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procSendInput        = user32.NewProc("SendInput")
	procVkKeyScan        = user32.NewProc("VkKeyScanW")
	procGetCursorInfo    = user32.NewProc("GetCursorInfo")
	procGetIconInfo      = user32.NewProc("GetIconInfo")

	// gdi32
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procCreateDIBSection       = gdi32.NewProc("CreateDIBSection")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procGetObject              = gdi32.NewProc("GetObjectW")
)

const (
	smCxScreen   = 0
	smCyScreen   = 1
	srcCopy      = 0x00CC0020
	dibRgbColors = 0
)

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}
