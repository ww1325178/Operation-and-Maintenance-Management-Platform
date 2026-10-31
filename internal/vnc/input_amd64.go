//go:build amd64

package vnc

// INPUT mirrors the Windows INPUT struct on 64-bit: 4 type + 4 padding +
// 32-byte union = 40 bytes.
type INPUT struct {
	Type uint32
	_    uint32
	Mi   MOUSEINPUT
}
