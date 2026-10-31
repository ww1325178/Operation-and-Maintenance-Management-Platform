package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"ops-platform/internal/protocol"
	"ops-platform/internal/vnc"
)

const Version = "0.1.0"

type Config struct {
	ServerURL string
	Token     string
	DeviceID  string
	Interval  time.Duration
}

type outMsg struct {
	mt   int
	data []byte
}

type vncSession struct {
	conn net.Conn
	ln   net.Listener
}

type Client struct {
	cfg   Config
	hello protocol.Hello

	conn *websocket.Conn
	wch  chan outMsg
	done chan struct{}

	mu       sync.Mutex
	sessions map[string]*vncSession
}

func Run(ctx context.Context, cfg Config) {
	hello := CollectInventory()
	hello.AgentVersion = Version

	backoff := time.Second
	for {
		ran, err := runOnce(ctx, cfg, hello)
		if err != nil && ctx.Err() == nil {
			log.Printf("连接服务器失败: %v", err)
		}
		if ran > 30*time.Second {
			backoff = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func runOnce(ctx context.Context, cfg Config, hello protocol.Hello) (time.Duration, error) {
	u, err := url.Parse(cfg.ServerURL)
	if err != nil {
		return 0, err
	}
	q := u.Query()
	q.Set("token", cfg.Token)
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	c := &Client{
		cfg:      cfg,
		hello:    hello,
		conn:     conn,
		wch:      make(chan outMsg, 1024),
		done:     make(chan struct{}),
		sessions: map[string]*vncSession{},
	}
	defer close(c.done)
	start := time.Now()

	if err := c.sendJSON(protocol.MsgHello, cfg.DeviceID, hello); err != nil {
		return time.Since(start), err
	}

	errCh := make(chan error, 3)
	go c.writeLoop(ctx, errCh)
	go c.heartbeatLoop(ctx, errCh)
	go c.readLoop(errCh)

	err = <-errCh
	c.closeAllSessions()
	return time.Since(start), err
}

func (c *Client) sendJSON(t string, deviceID string, payload any) error {
	b, err := protocol.Marshal(t, deviceID, payload)
	if err != nil {
		return err
	}
	select {
	case c.wch <- outMsg{websocket.TextMessage, b}:
		return nil
	default:
		return fmt.Errorf("写队列已满")
	}
}

func (c *Client) writeLoop(ctx context.Context, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-c.wch:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(m.mt, m.data); err != nil {
				errCh <- err
				return
			}
		}
	}
}

func (c *Client) heartbeatLoop(ctx context.Context, errCh chan<- error) {
	t := time.NewTicker(c.cfg.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := c.sendJSON(protocol.MsgHeartbeat, c.cfg.DeviceID,
				protocol.Heartbeat{Ts: time.Now().UnixMilli()}); err != nil {
				errCh <- err
				return
			}
		}
	}
}

func (c *Client) readLoop(errCh chan<- error) {
	for {
		mt, data, err := c.conn.ReadMessage()
		if err != nil {
			errCh <- err
			return
		}
		if mt == websocket.BinaryMessage {
			if sid, payload, ok := protocol.ParseStreamFrame(data); ok {
				c.onStream(sid, payload)
			}
			continue
		}
		var msg protocol.Message
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		c.onControl(msg)
	}
}

func (c *Client) onControl(msg protocol.Message) {
	switch msg.Type {
	case protocol.MsgStartVNC:
		var v protocol.StartVNC
		if protocol.UnmarshalData(msg, &v) != nil {
			return
		}
		c.startVNC(v.SessionID, v.Scale)
	case protocol.MsgStopVNC:
		var v protocol.StopVNC
		if protocol.UnmarshalData(msg, &v) != nil {
			return
		}
		c.stopVNC(v.SessionID)
	case protocol.MsgPing:
		// keep-alive
	}
}

// startVNC brings up an in-process VNC server on a loopback listener, then
// tunnels its bytes over the agent channel as binary stream frames.
func (c *Client) startVNC(sessionID string, scale float64) {
	log.Printf("[vnc] start session=%s scale=%v", sessionID, scale)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		c.sendReady(sessionID, false, err.Error())
		return
	}

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			_ = ln.Close()
			return
		}
		err = vnc.NewServer(scale).Serve(conn)
		log.Printf("[vnc] serve 结束 session=%s err=%v", sessionID, err)
		_ = conn.Close()
		_ = ln.Close()
	}()

	clientConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		_ = ln.Close()
		c.sendReady(sessionID, false, err.Error())
		return
	}

	s := &vncSession{conn: clientConn, ln: ln}
	c.mu.Lock()
	c.sessions[sessionID] = s
	c.mu.Unlock()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := clientConn.Read(buf)
			if n > 0 {
				frame := protocol.NewStreamFrame(sessionID, buf[:n])
				select {
				case c.wch <- outMsg{websocket.BinaryMessage, frame}:
				case <-c.done:
					return
				}
			}
			if err != nil {
				break
			}
		}
		c.stopVNC(sessionID)
	}()

	c.sendReady(sessionID, true, "")
}

func (c *Client) stopVNC(sessionID string) {
	log.Printf("[vnc] stop session=%s", sessionID)
	c.mu.Lock()
	s := c.sessions[sessionID]
	delete(c.sessions, sessionID)
	c.mu.Unlock()
	if s == nil {
		return
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
	if s.ln != nil {
		_ = s.ln.Close()
	}
}

func (c *Client) onStream(sessionID string, payload []byte) {
	c.mu.Lock()
	s := c.sessions[sessionID]
	c.mu.Unlock()
	if s != nil && s.conn != nil {
		_, _ = s.conn.Write(payload)
	}
}

func (c *Client) sendReady(sessionID string, ok bool, errMsg string) {
	_ = c.sendJSON(protocol.MsgVNCReady, c.cfg.DeviceID,
		protocol.VNCReady{SessionID: sessionID, OK: ok, Error: errMsg})
}

func (c *Client) closeAllSessions() {
	c.mu.Lock()
	ids := make([]string, 0, len(c.sessions))
	for id := range c.sessions {
		ids = append(ids, id)
	}
	c.mu.Unlock()
	for _, id := range ids {
		c.stopVNC(id)
	}
}
