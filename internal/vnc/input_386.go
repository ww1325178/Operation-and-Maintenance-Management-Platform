//go:build 386

package vnc

// INPUT mirrors the Windows INPUT struct on 32-bit: 4 type + 24-byte union =
// 28 bytes (no padding before the union on 32-bit).
type INPUT struct {
	Type uint32
	Mi   MOUSEINPUT
}
