package vnc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"sync"
)

const protocolVersion = "RFB 003.008\n"

// Server is a minimal RFB 3.8 server: raw encoding, security type None,
// 32bpp true colour, primary-monitor capture, absolute pointer + keyboard.
// A downscale factor (1/2/4) reduces the transmitted resolution.
type Server struct {
	mu     sync.Mutex
	factor int
	w, h   int // real screen size
	fw, fh int // framebuffer (downscaled) size
	last   []byte
	in     *Input
}

// NewServer builds a server with the given resolution scale (1.0 / 0.5 / 0.25).
func NewServer(scale float64) *Server {
	f := 1
	switch {
	case scale <= 0.4:
		f = 4
	case scale <= 0.75:
		f = 2
	}
	return &Server{factor: f}
}

func (s *Server) Serve(conn net.Conn) error {
	if _, err := conn.Write([]byte(protocolVersion)); err != nil {
		return err
	}
	if _, err := io.ReadFull(conn, make([]byte, 12)); err != nil {
		return err
	}
	// one security type: None (1)
	if _, err := conn.Write([]byte{1, 1}); err != nil {
		return err
	}
	var one [1]byte
	if _, err := io.ReadFull(conn, one[:]); err != nil {
		return err
	}
	// SecurityResult: 4-byte big-endian status (0 = OK). noVNC expects this
	// even for the "None" type on RFB 3.8.
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return err
	}
	// ClientInit shared flag
	if _, err := io.ReadFull(conn, one[:]); err != nil {
		return err
	}

	w, h := screenSize()
	if w <= 0 || h <= 0 {
		return errors.New("screen size invalid")
	}
	s.mu.Lock()
	s.w, s.h = w, h
	s.fw, s.fh = w/s.factor, h/s.factor
	s.in = NewInput(w, h, s.factor)
	s.mu.Unlock()

	if err := s.writeServerInit(conn); err != nil {
		return err
	}
	return s.loop(conn)
}

func (s *Server) writeServerInit(w io.Writer) error {
	s.mu.Lock()
	fw, fh := s.fw, s.fh
	s.mu.Unlock()

	var buf [24]byte
	binary.BigEndian.PutUint16(buf[0:], uint16(fw))
	binary.BigEndian.PutUint16(buf[2:], uint16(fh))
	buf[4] = 32 // bits-per-pixel
	buf[5] = 24 // depth
	buf[6] = 0  // big-endian flag
	buf[7] = 1  // true colour
	binary.BigEndian.PutUint16(buf[8:], 255)  // red max
	binary.BigEndian.PutUint16(buf[10:], 255) // green max
	binary.BigEndian.PutUint16(buf[12:], 255) // blue max
	buf[14] = 16 // red shift
	buf[15] = 8  // green shift
	buf[16] = 0  // blue shift
	buf[17], buf[18], buf[19] = 0, 0, 0
	name := "ops-agent"
	binary.BigEndian.PutUint32(buf[20:], uint32(len(name)))

	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	_, err := w.Write([]byte(name))
	return err
}

func (s *Server) loop(conn net.Conn) error {
	var header [1]byte
	for {
		if _, err := io.ReadFull(conn, header[:]); err != nil {
			return err
		}
		switch header[0] {
		case 0: // SetPixelFormat
			if _, err := io.ReadFull(conn, make([]byte, 19)); err != nil {
				return err
			}
		case 2: // SetEncodings: 1 pad + uint16 count + count*uint32 encodings
			var b [3]byte
			if _, err := io.ReadFull(conn, b[:]); err != nil {
				return err
			}
			n := binary.BigEndian.Uint16(b[1:])
			if _, err := io.ReadFull(conn, make([]byte, int(n)*4)); err != nil {
				return err
			}
		case 3: // FramebufferUpdateRequest
			var req [9]byte
			if _, err := io.ReadFull(conn, req[:]); err != nil {
				return err
			}
			if err := s.sendUpdate(conn, req[0] != 0); err != nil {
				return err
			}
		case 4: // KeyEvent
			var e [7]byte
			if _, err := io.ReadFull(conn, e[:]); err != nil {
				return err
			}
			key := binary.BigEndian.Uint32(e[3:])
			log.Printf("[vnc] key keysym=0x%x down=%v", key, e[0] != 0)
			s.mu.Lock()
			in := s.in
			s.mu.Unlock()
			if in != nil {
				in.Key(key, e[0] != 0)
			}
		case 5: // PointerEvent
			var e [5]byte
			if _, err := io.ReadFull(conn, e[:]); err != nil {
				return err
			}
			px := binary.BigEndian.Uint16(e[1:])
			py := binary.BigEndian.Uint16(e[3:])
			s.mu.Lock()
			in := s.in
			s.mu.Unlock()
			if in != nil {
				in.Pointer(e[0], px, py)
			}
		case 6: // ClientCutText
			var b [7]byte
			if _, err := io.ReadFull(conn, b[:]); err != nil {
				return err
			}
			n := binary.BigEndian.Uint32(b[3:])
			if _, err := io.ReadFull(conn, make([]byte, n)); err != nil {
				return err
			}
		case 150: // EnableContinuousUpdates: enable + x + y + w + h
			if _, err := io.ReadFull(conn, make([]byte, 9)); err != nil {
				return err
			}
		case 248: // ClientFence: 3 pad + uint32 flags + uint8 len + payload
			var b [8]byte
			if _, err := io.ReadFull(conn, b[:]); err != nil {
				return err
			}
			if _, err := io.ReadFull(conn, make([]byte, int(b[7]))); err != nil {
				return err
			}
		case 250: // xvpOp: pad + version + op
			if _, err := io.ReadFull(conn, make([]byte, 3)); err != nil {
				return err
			}
		case 251: // SetDesktopSize: pad + w + h + num + pad + num*16
			var b [7]byte
			if _, err := io.ReadFull(conn, b[:]); err != nil {
				return err
			}
			if _, err := io.ReadFull(conn, make([]byte, int(b[5])*16)); err != nil {
				return err
			}
		case 255: // QEMU extended key event: sub + down + keysym + keycode
			if _, err := io.ReadFull(conn, make([]byte, 11)); err != nil {
				return err
			}
		default:
			log.Printf("[vnc] 未知消息类型 %d，断开", header[0])
			return nil
		}
	}
}

func (s *Server) sendUpdate(conn net.Conn, incremental bool) error {
	fb, w, h, err := captureScreen()
	if err != nil {
		log.Printf("[vnc] 屏幕采集失败: %v", err)
		return err
	}
	if s.factor > 1 {
		fb = downscale(fb, w, h, s.factor)
		w, h = w/s.factor, h/s.factor
	}

	s.mu.Lock()
	s.fw, s.fh = w, h
	last := s.last
	if !incremental || last == nil || len(last) != len(fb) {
		if err := s.sendFull(conn, fb, w, h); err != nil {
			s.mu.Unlock()
			return err
		}
		s.last = fb
		s.mu.Unlock()
		return nil
	}
	rects := diffRects(last, fb, w, h)
	s.last = fb
	s.mu.Unlock()
	return s.sendRects(conn, fb, w, rects)
}

func (s *Server) sendFull(conn net.Conn, fb []byte, w, h int) error {
	if _, err := conn.Write([]byte{0, 0, 0, 1}); err != nil { // type 0, pad 0, 1 rect
		return err
	}
	rect := make([]byte, 12)
	binary.BigEndian.PutUint16(rect[0:], 0)
	binary.BigEndian.PutUint16(rect[2:], 0)
	binary.BigEndian.PutUint16(rect[4:], uint16(w))
	binary.BigEndian.PutUint16(rect[6:], uint16(h))
	binary.BigEndian.PutUint32(rect[8:], 0) // raw encoding
	if _, err := conn.Write(rect); err != nil {
		return err
	}
	_, err := conn.Write(fb)
	return err
}

func (s *Server) sendRects(conn net.Conn, fb []byte, w int, rects []rect) error {
	hdr := make([]byte, 4)
	hdr[0] = 0
	hdr[1] = 0
	binary.BigEndian.PutUint16(hdr[2:], uint16(len(rects)))
	if _, err := conn.Write(hdr); err != nil {
		return err
	}
	for _, r := range rects {
		var rh [12]byte
		binary.BigEndian.PutUint16(rh[0:], uint16(r.x))
		binary.BigEndian.PutUint16(rh[2:], uint16(r.y))
		binary.BigEndian.PutUint16(rh[4:], uint16(r.w))
		binary.BigEndian.PutUint16(rh[6:], uint16(r.h))
		binary.BigEndian.PutUint32(rh[8:], 0)
		if _, err := conn.Write(rh[:]); err != nil {
			return err
		}
		for row := 0; row < int(r.h); row++ {
			off := (int(r.y)+row)*w*4 + int(r.x)*4
			if _, err := conn.Write(fb[off : off+int(r.w)*4]); err != nil {
				return err
			}
		}
	}
	return nil
}

type rect struct{ x, y, w, h int32 }

func diffRects(prev, cur []byte, w, h int) []rect {
	var runs []rect
	rowBytes := w * 4
	for y := 0; y < h; y++ {
		base := y * rowBytes
		if bytes.Equal(prev[base:base+rowBytes], cur[base:base+rowBytes]) {
			continue
		}
		x := 0
		for x < w {
			for x < w && equal32(prev, cur, base+x*4) {
				x++
			}
			if x >= w {
				break
			}
			xs := x
			for x < w && !equal32(prev, cur, base+x*4) {
				x++
			}
			runs = append(runs, rect{int32(xs), int32(y), int32(x - xs), 1})
		}
	}
	return coalesce(runs)
}

func equal32(a, b []byte, off int) bool {
	// 忽略第 4 字节（32bpp 的 alpha 未定义，逐次捕获会抖动），只比较 RGB
	return a[off] == b[off] && a[off+1] == b[off+1] && a[off+2] == b[off+2]
}

func coalesce(rs []rect) []rect {
	var out []rect
	for _, r := range rs {
		if n := len(out); n > 0 && out[n-1].x == r.x && out[n-1].w == r.w && out[n-1].y+out[n-1].h == r.y {
			out[n-1].h++
		} else {
			out = append(out, r)
		}
	}
	return out
}

// downscale performs box-filter downsampling by an integer factor (2 or 4).
func downscale(src []byte, sw, sh, factor int) []byte {
	dw, dh := sw/factor, sh/factor
	dst := make([]byte, dw*dh*4)
	n := factor * factor
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			base := (y*factor*sw + x*factor) * 4
			var r, g, b int
			for dy := 0; dy < factor; dy++ {
				row := base + dy*sw*4
				for dx := 0; dx < factor; dx++ {
					off := row + dx*4
					b += int(src[off])
					g += int(src[off+1])
					r += int(src[off+2])
				}
			}
			off := (y*dw + x) * 4
			dst[off] = byte(b / n)
			dst[off+1] = byte(g / n)
			dst[off+2] = byte(r / n)
			dst[off+3] = 0xff
		}
	}
	return dst
}
