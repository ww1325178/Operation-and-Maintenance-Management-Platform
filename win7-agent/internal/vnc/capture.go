package vnc

import (
	"errors"
	"unsafe"
)

func screenSize() (int, int) {
	w, _, _ := procGetSystemMetrics.Call(smCxScreen)
	h, _, _ := procGetSystemMetrics.Call(smCyScreen)
	return int(w), int(h)
}

// captureScreen grabs the primary monitor as a 32bpp BGRA buffer (top-down).
// It uses CreateDIBSection + BitBlt and reads the pixel memory directly, which
// avoids GetDIBits (that API fails on some display drivers, notably VMs on
// Windows 7).
func captureScreen() ([]byte, int, int, error) {
	w, h := screenSize()
	if w <= 0 || h <= 0 {
		return nil, 0, 0, errors.New("screen size invalid")
	}

	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return nil, 0, 0, errors.New("GetDC failed")
	}
	defer procReleaseDC.Call(0, hdc)

	memDC, _, _ := procCreateCompatibleDC.Call(hdc)
	if memDC == 0 {
		return nil, 0, 0, errors.New("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(memDC)

	bmi := struct {
		header bitmapInfoHeader
		colors [1]uint32
	}{
		header: bitmapInfoHeader{
			Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			Width:       int32(w),
			Height:      -int32(h), // negative => top-down rows
			Planes:      1,
			BitCount:    32,
			Compression: 0, // BI_RGB
		},
	}

	var bits uintptr
	hbmp, _, _ := procCreateDIBSection.Call(memDC, uintptr(unsafe.Pointer(&bmi)),
		dibRgbColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbmp == 0 {
		return nil, 0, 0, errors.New("CreateDIBSection failed")
	}
	defer procDeleteObject.Call(hbmp)

	old, _, _ := procSelectObject.Call(memDC, hbmp)
	defer procSelectObject.Call(memDC, old)

	if r, _, _ := procBitBlt.Call(memDC, 0, 0, uintptr(w), uintptr(h), hdc, 0, 0, srcCopy); r == 0 {
		return nil, 0, 0, errors.New("BitBlt failed")
	}

	buf := make([]byte, w*h*4)
	copy(buf, unsafe.Slice((*byte)(unsafe.Pointer(bits)), w*h*4))
	drawCursor(buf, w, h)
	return buf, w, h, nil
}
