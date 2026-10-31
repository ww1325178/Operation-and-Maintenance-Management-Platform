package vnc

import (
	"testing"
	"unsafe"
)

type pt struct{ x, y int32 }

func getCursorPos() (int32, int32) {
	var p pt
	proc := user32.NewProc("GetCursorPos")
	proc.Call(uintptr(unsafe.Pointer(&p)))
	return p.x, p.y
}

func TestSendInputMovesCursor(t *testing.T) {
	w, h := screenSize()
	beforeX, beforeY := getCursorPos()

	in := NewInput(w, h, 1)
	// 移动鼠标到屏幕中心
	in.Pointer(0, uint16(w/2), uint16(h/2))

	afterX, afterY := getCursorPos()
	t.Logf("屏幕 %dx%d  鼠标: before=(%d,%d) -> after=(%d,%d)", w, h, beforeX, beforeY, afterX, afterY)

	if beforeX == afterX && beforeY == afterY {
		t.Fatalf("SendInput 没有移动鼠标（可能被拦截）")
	}
}

func TestSendInputKey(t *testing.T) {
	in := NewInput(1920, 1080, 1)
	in.Key('a', true)
	in.Key('a', false)
	t.Log("键盘注入已调用（需人工确认是否生效）")
}
