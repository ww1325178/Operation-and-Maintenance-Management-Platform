package vnc

import (
	"errors"
	"log"
	"unsafe"
)

// Windows SendInput structs (x64 layout). The union is 32 bytes wide.
type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

const (
	inputMouse    = 0
	inputKeyboard = 1

	keyeventfKeyup = 0x0002

	mouseeventfMove       = 0x0001
	mouseeventfLeftdown   = 0x0002
	mouseeventfLeftup     = 0x0004
	mouseeventfRightdown  = 0x0008
	mouseeventfRightup    = 0x0010
	mouseeventfMiddledown = 0x0020
	mouseeventfMiddleup   = 0x0040
	mouseeventfWheel      = 0x0800
	mouseeventfAbsolute   = 0x8000

	wheelDelta     = 120
	wheelDeltaDown = uint32(0xFFFFFF88) // (DWORD)-WHEEL_DELTA
)

func newKeyInput(ki KEYBDINPUT) INPUT {
	var in INPUT
	in.Type = inputKeyboard
	*(*KEYBDINPUT)(unsafe.Pointer(&in.Mi)) = ki
	return in
}

func newMouseInput(mi MOUSEINPUT) INPUT {
	var in INPUT
	in.Type = inputMouse
	in.Mi = mi
	return in
}

func sendInput(inputs []INPUT) error {
	if len(inputs) == 0 {
		return nil
	}
	n, _, _ := procSendInput.Call(uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(INPUT{}))
	if int(n) != len(inputs) {
		log.Printf("[vnc] SendInput 失败: 注入 %d/%d（可能被 UAC/权限拦截）", n, len(inputs))
		return errors.New("SendInput failed")
	}
	return nil
}

// Input translates RFB key/pointer events into Windows input.
type Input struct {
	w, h     int
	factor   int
	lastMask uint8
}

func NewInput(w, h, factor int) *Input {
	if factor < 1 {
		factor = 1
	}
	return &Input{w: w, h: h, factor: factor}
}

func (in *Input) Key(keysym uint32, down bool) {
	vk, needsShift := keysymToVK(keysym)
	if vk == 0 {
		return
	}
	if needsShift {
		if down {
			_ = sendInput([]INPUT{newKeyInput(KEYBDINPUT{WVk: 0x10})})
			_ = sendInput([]INPUT{newKeyInput(KEYBDINPUT{WVk: vk})})
		} else {
			_ = sendInput([]INPUT{newKeyInput(KEYBDINPUT{WVk: vk, DwFlags: keyeventfKeyup})})
			_ = sendInput([]INPUT{newKeyInput(KEYBDINPUT{WVk: 0x10, DwFlags: keyeventfKeyup})})
		}
		return
	}
	flags := uint32(0)
	if !down {
		flags = keyeventfKeyup
	}
	_ = sendInput([]INPUT{newKeyInput(KEYBDINPUT{WVk: vk, DwFlags: flags})})
}

func (in *Input) Pointer(mask uint8, x, y uint16) {
	realX := int32(x) * int32(in.factor)
	realY := int32(y) * int32(in.factor)
	var mx, my int32
	if in.w > 1 {
		mx = realX * 65535 / int32(in.w-1)
	}
	if in.h > 1 {
		my = realY * 65535 / int32(in.h-1)
	}

	_ = sendInput([]INPUT{newMouseInput(MOUSEINPUT{
		Dx: mx, Dy: my, DwFlags: mouseeventfMove | mouseeventfAbsolute,
	})})

	prev := in.lastMask
	in.lastMask = mask
	changed := prev ^ mask

	if changed&0x01 != 0 {
		flag := uint32(mouseeventfLeftup)
		if mask&0x01 != 0 {
			flag = mouseeventfLeftdown
		}
		_ = sendInput([]INPUT{newMouseInput(MOUSEINPUT{DwFlags: flag})})
	}
	if changed&0x02 != 0 {
		flag := uint32(mouseeventfMiddleup)
		if mask&0x02 != 0 {
			flag = mouseeventfMiddledown
		}
		_ = sendInput([]INPUT{newMouseInput(MOUSEINPUT{DwFlags: flag})})
	}
	if changed&0x04 != 0 {
		flag := uint32(mouseeventfRightup)
		if mask&0x04 != 0 {
			flag = mouseeventfRightdown
		}
		_ = sendInput([]INPUT{newMouseInput(MOUSEINPUT{DwFlags: flag})})
	}
	if mask&0x08 != 0 {
		_ = sendInput([]INPUT{newMouseInput(MOUSEINPUT{MouseData: wheelDelta, DwFlags: mouseeventfWheel})})
	}
	if mask&0x10 != 0 {
		_ = sendInput([]INPUT{newMouseInput(MOUSEINPUT{MouseData: wheelDeltaDown, DwFlags: mouseeventfWheel})})
	}
}

var specialKeys = map[uint32]uint16{
	0xFF08: 0x08, // BackSpace
	0xFF09: 0x09, // Tab
	0xFF0D: 0x0D, // Return
	0xFF1B: 0x1B, // Escape
	0xFF63: 0x2D, // Insert
	0xFFFF: 0x2E, // Delete
	0xFF50: 0x24, // Home
	0xFF57: 0x23, // End
	0xFF55: 0x21, // PageUp
	0xFF56: 0x22, // PageDown
	0xFF51: 0x25, // Left
	0xFF52: 0x26, // Up
	0xFF53: 0x27, // Right
	0xFF54: 0x28, // Down
	0xFFBE: 0x70, // F1
	0xFFBF: 0x71, // F2
	0xFFC0: 0x72, // F3
	0xFFC1: 0x73, // F4
	0xFFC2: 0x74, // F5
	0xFFC3: 0x75, // F6
	0xFFC4: 0x76, // F7
	0xFFC5: 0x77, // F8
	0xFFC6: 0x78, // F9
	0xFFC7: 0x79, // F10
	0xFFC8: 0x7A, // F11
	0xFFC9: 0x7B, // F12
	0xFFE1: 0x10, // Shift_L
	0xFFE2: 0x10, // Shift_R
	0xFFE3: 0x11, // Control_L
	0xFFE4: 0x11, // Control_R
	0xFFE9: 0x12, // Alt_L
	0xFFEA: 0x12, // Alt_R
	0xFFEB: 0x5B, // Super_L (Win)
	0xFFEC: 0x5C, // Super_R
	0x0020: 0x20, // space
}

func keysymToVK(k uint32) (vk uint16, needsShift bool) {
	if v, ok := specialKeys[k]; ok {
		return v, false
	}
	if k >= 0x20 && k <= 0x7E {
		switch {
		case k >= 'a' && k <= 'z':
			return uint16(k - 0x20), false
		case k >= 'A' && k <= 'Z':
			return uint16(k), false
		case k >= '0' && k <= '9':
			return uint16(k), false
		default:
			r, _, _ := procVkKeyScan.Call(uintptr(k))
			res := int16(r & 0xFFFF)
			if res == -1 {
				return 0, false
			}
			return uint16(res & 0xFF), (res>>8)&1 == 1
		}
	}
	return 0, false
}
