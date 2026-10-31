package vnc

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
)

func TestHandshakeAndFullUpdate(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go func() { _ = NewServer(1.0).Serve(serverConn) }()

	// 1. server version
	ver := make([]byte, 12)
	if _, err := io.ReadFull(clientConn, ver); err != nil {
		t.Fatal(err)
	}
	if string(ver) != "RFB 003.008\n" {
		t.Fatalf("version = %q", ver)
	}
	// 2. client version
	if _, err := clientConn.Write([]byte("RFB 003.008\n")); err != nil {
		t.Fatal(err)
	}
	// 3. security types
	st := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, st); err != nil {
		t.Fatal(err)
	}
	if st[0] != 1 || st[1] != 1 {
		t.Fatalf("security types = %v", st)
	}
	// 4. choose None
	if _, err := clientConn.Write([]byte{1}); err != nil {
		t.Fatal(err)
	}
	// 4b. SecurityResult (4 bytes, 0 = OK)
	sr := make([]byte, 4)
	if _, err := io.ReadFull(clientConn, sr); err != nil {
		t.Fatal(err)
	}
	if binary.BigEndian.Uint32(sr) != 0 {
		t.Fatalf("security result = %d", binary.BigEndian.Uint32(sr))
	}
	// 4c. ClientInit shared flag
	if _, err := clientConn.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}
	// 5. server init
	hdr := make([]byte, 24)
	if _, err := io.ReadFull(clientConn, hdr); err != nil {
		t.Fatal(err)
	}
	w := binary.BigEndian.Uint16(hdr[0:])
	h := binary.BigEndian.Uint16(hdr[2:])
	if w == 0 || h == 0 {
		t.Fatalf("bad dims %dx%d", w, h)
	}
	name := make([]byte, binary.BigEndian.Uint32(hdr[20:]))
	if _, err := io.ReadFull(clientConn, name); err != nil {
		t.Fatal(err)
	}

	// 6. SetEncodings (raw only)
	if _, err := clientConn.Write([]byte{2, 0, 0, 1, 0, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
	// 7. FramebufferUpdateRequest (full)
	req := make([]byte, 10)
	req[0] = 3
	binary.BigEndian.PutUint16(req[2:], 0)
	binary.BigEndian.PutUint16(req[4:], 0)
	binary.BigEndian.PutUint16(req[6:], w)
	binary.BigEndian.PutUint16(req[8:], h)
	if _, err := clientConn.Write(req); err != nil {
		t.Fatal(err)
	}

	// 8. FramebufferUpdate response
	var up [4]byte
	if _, err := io.ReadFull(clientConn, up[:]); err != nil {
		t.Fatal(err)
	}
	if up[0] != 0 {
		t.Fatalf("msg type = %d", up[0])
	}
	nrect := binary.BigEndian.Uint16(up[2:])
	if nrect != 1 {
		t.Fatalf("num rects = %d", nrect)
	}
	var r [12]byte
	if _, err := io.ReadFull(clientConn, r[:]); err != nil {
		t.Fatal(err)
	}
	rw := binary.BigEndian.Uint16(r[4:])
	rh := binary.BigEndian.Uint16(r[6:])
	enc := binary.BigEndian.Uint32(r[8:])
	if enc != 0 {
		t.Fatalf("encoding = %d", enc)
	}
	if int(rw) != int(w) || int(rh) != int(h) {
		t.Fatalf("rect %dx%d vs screen %dx%d", rw, rh, w, h)
	}
	px := make([]byte, int(rw)*int(rh)*4)
	if _, err := io.ReadFull(clientConn, px); err != nil {
		t.Fatalf("pixels: %v", err)
	}
	first := binary.LittleEndian.Uint32(px[0:])
	varied := false
	for i := 1; i < len(px)/4; i++ {
		if binary.LittleEndian.Uint32(px[i*4:]) != first {
			varied = true
			break
		}
	}
	t.Logf("handshake ok: %dx%d name=%q enc=%d pixels=%d first=0x%08x varied=%v", w, h, name, enc, len(px), first, varied)
	if !varied {
		t.Fatalf("captured framebuffer is blank (all pixels identical)")
	}
}
