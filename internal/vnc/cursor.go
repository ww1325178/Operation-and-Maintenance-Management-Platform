package vnc

import (
	"unsafe"
)

type point struct{ X, Y int32 }

type cursorInfo struct {
	CbSize      uint32
	Flags       uint32
	HCursor     uintptr
	PtScreenPos point
}

type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

type gdiBitmap struct {
	BmType       int32
	BmWidth      int32
	BmHeight     int32
	BmWidthBytes int32
	BmPlanes     uint16
	BmBitsPixel  uint16
	BmBits       uintptr
}

const cursorShowing = 0x00000001

// drawCursor 把当前鼠标光标合成到屏幕帧上（GDI 采集不含光标，需单独画）。
func drawCursor(fb []byte, w, h int) {
	var ci cursorInfo
	ci.CbSize = uint32(unsafe.Sizeof(cursorInfo{}))
	r, _, _ := procGetCursorInfo.Call(uintptr(unsafe.Pointer(&ci)))
	if r == 0 || ci.Flags&cursorShowing == 0 || ci.HCursor == 0 {
		return
	}

	var ii iconInfo
	if r, _, _ := procGetIconInfo.Call(ci.HCursor, uintptr(unsafe.Pointer(&ii))); r == 0 {
		return
	}
	defer procDeleteObject.Call(ii.HbmMask)
	if ii.HbmColor != 0 {
		defer procDeleteObject.Call(ii.HbmColor)
	}
	if ii.HbmColor == 0 {
		return // 单色光标暂不处理
	}

	var bm gdiBitmap
	if n, _, _ := procGetObject.Call(ii.HbmColor, uintptr(unsafe.Sizeof(gdiBitmap{})), uintptr(unsafe.Pointer(&bm))); n == 0 {
		return
	}
	cw, ch := int(bm.BmWidth), int(bm.BmHeight)
	if cw <= 0 || ch <= 0 || cw > 512 || ch > 512 {
		return
	}

	cpix := bitmapPixels(ii.HbmColor, cw, ch)
	if cpix == nil {
		return
	}

	dx := int(ci.PtScreenPos.X) - int(ii.XHotspot)
	dy := int(ci.PtScreenPos.Y) - int(ii.YHotspot)

	for y := 0; y < ch; y++ {
		sy := dy + y
		if sy < 0 || sy >= h {
			continue
		}
		for x := 0; x < cw; x++ {
			sx := dx + x
			if sx < 0 || sx >= w {
				continue
			}
			so := (y*cw + x) * 4
			a := int(cpix[so+3])
			if a == 0 {
				continue
			}
			do := (sy*w + sx) * 4
			if a == 255 {
				fb[do] = cpix[so]
				fb[do+1] = cpix[so+1]
				fb[do+2] = cpix[so+2]
			} else {
				na := 255 - a
				fb[do] = byte((int(fb[do])*na + int(cpix[so])*a) / 255)
				fb[do+1] = byte((int(fb[do+1])*na + int(cpix[so+1])*a) / 255)
				fb[do+2] = byte((int(fb[do+2])*na + int(cpix[so+2])*a) / 255)
			}
		}
	}
}

// bitmapPixels 用 GetDIBits 获取位图的 32bpp BGRA 像素（含 alpha）。光标位图是
// DIB section，GetDIBits 在此处可用（屏幕采集那个 DDB 才需要绕开）。
func bitmapPixels(hbm uintptr, w, h int) []byte {
	memDC, _, _ := procCreateCompatibleDC.Call(0)
	if memDC == 0 {
		return nil
	}
	defer procDeleteDC.Call(memDC)

	old, _, _ := procSelectObject.Call(memDC, hbm)
	defer procSelectObject.Call(memDC, old)

	bmi := struct {
		header bitmapInfoHeader
		colors [1]uint32
	}{
		header: bitmapInfoHeader{
			Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			Width:       int32(w),
			Height:      -int32(h), // 输出 top-down
			Planes:      1,
			BitCount:    32,
			Compression: 0,
		},
	}

	buf := make([]byte, w*h*4)
	if r, _, _ := procGetDIBits.Call(memDC, hbm, 0, uintptr(h),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bmi)), dibRgbColors); r == 0 {
		return nil
	}
	return buf
}
