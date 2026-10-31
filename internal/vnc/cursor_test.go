package vnc

import (
	"testing"
	"unsafe"
)

func TestCursorRetrieval(t *testing.T) {
	var ci cursorInfo
	ci.CbSize = uint32(unsafe.Sizeof(cursorInfo{}))
	r, _, _ := procGetCursorInfo.Call(uintptr(unsafe.Pointer(&ci)))
	if r == 0 || ci.HCursor == 0 {
		t.Fatal("GetCursorInfo 失败")
	}
	var ii iconInfo
	if r, _, _ := procGetIconInfo.Call(ci.HCursor, uintptr(unsafe.Pointer(&ii))); r == 0 {
		t.Fatal("GetIconInfo 失败")
	}
	defer procDeleteObject.Call(ii.HbmMask)
	if ii.HbmColor != 0 {
		defer procDeleteObject.Call(ii.HbmColor)
	}

	t.Logf("光标 flags=%d hbmColor=%v hbmMask=%v 热点=(%d,%d) 位置=(%d,%d)",
		ci.Flags, ii.HbmColor, ii.HbmMask, ii.XHotspot, ii.YHotspot, ci.PtScreenPos.X, ci.PtScreenPos.Y)

	if ii.HbmColor == 0 {
		t.Log("单色光标（暂不处理）")
		return
	}
	var bm gdiBitmap
	if n, _, _ := procGetObject.Call(ii.HbmColor, uintptr(unsafe.Sizeof(gdiBitmap{})), uintptr(unsafe.Pointer(&bm))); n == 0 {
		t.Fatal("GetObject 失败")
	}
	t.Logf("光标位图 %dx%d bpp=%d", bm.BmWidth, bm.BmHeight, bm.BmBitsPixel)

	cpix := bitmapPixels(ii.HbmColor, int(bm.BmWidth), int(bm.BmHeight))
	if cpix == nil {
		t.Fatal("bitmapPixels 失败（GetDIBits 失败）")
	}
	opaque := 0
	for i := 3; i < len(cpix); i += 4 {
		if cpix[i] != 0 {
			opaque++
		}
	}
	t.Logf("光标像素 %d 个，非透明 %d 个", len(cpix)/4, opaque)
	if opaque == 0 {
		t.Fatal("光标无有效像素")
	}
}
