package common

// Real client addresses behind rootless port publishing (prds#149). Podman's rootless port
// forwarder rewrites the source of every connection and datagram to a container-network
// address, so QR2, NATNEG, GPCM and the server browser saw 10.89.x.x instead of a player's
// public address: QR2 dropped Wii heartbeats ("Public IP mismatch") and NATNEG could not tell
// peers each other's addresses. wfc-front (a pasta-networked relay, like openpak-edge) receives
// the public ports with the real addresses and passes them on with PROXY protocol v2:
//
//   - TCP: one v2 header (PROXY, STREAM) at the start of each connection, to a second,
//     loopback-only listener of the service (ProxyListener);
//   - UDP: every datagram framed with a v2 header (PROXY, DGRAM). Client to service, the source
//     is the client; service to client, the destination is the client, so the service can also
//     send unsolicited datagrams (QR2 to a host, NATNEG to both peers) through the relay
//     (DualPacketConn).
//
// The direct ports keep working unchanged, so a service runs the same before and after the
// public ports are switched to the relay.

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"
)

var proxySig = []byte("\r\n\r\n\x00\r\nQUIT\n")

const (
	proxyStream = 0x1
	proxyDgram  = 0x2
)

// ErrNotProxy is a buffer or stream that does not start with a usable PROXY v2 header.
var ErrNotProxy = errors.New("not a PROXY v2 header")

// EncodeProxyV2 is a PROXY v2 header (command PROXY) for src -> dst over TCP (stream) or UDP.
func EncodeProxyV2(stream bool, srcIP net.IP, srcPort int, dstIP net.IP, dstPort int) []byte {
	out := append([]byte(nil), proxySig...)
	proto := byte(proxyDgram)
	if stream {
		proto = proxyStream
	}
	s4, d4 := srcIP.To4(), dstIP.To4()
	if s4 != nil && d4 != nil {
		out = append(out, 0x21, 0x10|proto)
		out = binary.BigEndian.AppendUint16(out, 12)
		out = append(out, s4...)
		out = append(out, d4...)
	} else {
		out = append(out, 0x21, 0x20|proto)
		out = binary.BigEndian.AppendUint16(out, 36)
		out = append(out, srcIP.To16()...)
		out = append(out, dstIP.To16()...)
	}
	out = binary.BigEndian.AppendUint16(out, uint16(srcPort))
	out = binary.BigEndian.AppendUint16(out, uint16(dstPort))
	return out
}

// ProxyV2Header is a parsed header.
type ProxyV2Header struct {
	SrcIP, DstIP     net.IP
	SrcPort, DstPort int
	Local            bool // command LOCAL: no addresses (health checks)
}

// ParseProxyV2 reads one header from the start of b and returns it with the bytes after it.
func ParseProxyV2(b []byte) (ProxyV2Header, []byte, error) {
	if len(b) < 16 || !bytes.Equal(b[:12], proxySig) || b[12]>>4 != 2 {
		return ProxyV2Header{}, nil, ErrNotProxy
	}
	n := int(binary.BigEndian.Uint16(b[14:16]))
	if len(b) < 16+n {
		return ProxyV2Header{}, nil, ErrNotProxy
	}
	h, rest := ProxyV2Header{}, b[16+n:]
	switch b[12] & 0xf {
	case 0: // LOCAL
		h.Local = true
		return h, rest, nil
	case 1: // PROXY
	default:
		return ProxyV2Header{}, nil, ErrNotProxy
	}
	a := b[16 : 16+n]
	switch b[13] >> 4 {
	case 1:
		if len(a) < 12 {
			return ProxyV2Header{}, nil, ErrNotProxy
		}
		h.SrcIP, h.DstIP = net.IP(append([]byte(nil), a[0:4]...)), net.IP(append([]byte(nil), a[4:8]...))
		h.SrcPort, h.DstPort = int(binary.BigEndian.Uint16(a[8:10])), int(binary.BigEndian.Uint16(a[10:12]))
	case 2:
		if len(a) < 36 {
			return ProxyV2Header{}, nil, ErrNotProxy
		}
		h.SrcIP, h.DstIP = net.IP(append([]byte(nil), a[0:16]...)), net.IP(append([]byte(nil), a[16:32]...))
		h.SrcPort, h.DstPort = int(binary.BigEndian.Uint16(a[32:34])), int(binary.BigEndian.Uint16(a[34:36]))
	default:
		return ProxyV2Header{}, nil, ErrNotProxy
	}
	return h, rest, nil
}

// readProxyV2 reads exactly one header from a stream.
func readProxyV2(br *bufio.Reader) (ProxyV2Header, error) {
	head, err := br.Peek(16)
	if err != nil {
		return ProxyV2Header{}, err
	}
	if !bytes.Equal(head[:12], proxySig) {
		return ProxyV2Header{}, ErrNotProxy
	}
	n := 16 + int(binary.BigEndian.Uint16(head[14:16]))
	if n > 16+216 {
		return ProxyV2Header{}, ErrNotProxy
	}
	full, err := br.Peek(n)
	if err != nil {
		return ProxyV2Header{}, err
	}
	h, _, err := ParseProxyV2(full)
	if err != nil {
		return ProxyV2Header{}, err
	}
	_, _ = br.Discard(n)
	return h, nil
}

// proxiedConn is a TCP connection whose remote address came from a PROXY header.
type proxiedConn struct {
	net.Conn
	br     *bufio.Reader
	remote net.Addr
}

func (c *proxiedConn) Read(p []byte) (int, error) { return c.br.Read(p) }
func (c *proxiedConn) RemoteAddr() net.Addr       { return c.remote }

// SetKeepAlive passes through to the TCP connection (the frontends set it).
func (c *proxiedConn) SetKeepAlive(on bool) error {
	if tc, ok := c.Conn.(*net.TCPConn); ok {
		return tc.SetKeepAlive(on)
	}
	return nil
}

// ProxyListener wraps a listener whose every connection starts with a PROXY v2 header (from
// wfc-front). Accept returns connections that report the client's address. A connection
// without a valid header within 5 seconds is closed and not returned.
type ProxyListener struct {
	net.Listener
}

// Accept implements net.Listener.
func (l ProxyListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
		br := bufio.NewReader(c)
		h, err := readProxyV2(br)
		_ = c.SetReadDeadline(time.Time{})
		if err != nil || h.Local {
			_ = c.Close()
			continue
		}
		return &proxiedConn{Conn: c, br: br, remote: &net.TCPAddr{IP: h.SrcIP, Port: h.SrcPort}}, nil
	}
}

// DualPacketConn is a game server's UDP socket: datagrams straight from players on the
// direct socket, and PROXY-framed datagrams from wfc-front on the proxy socket. ReadFrom
// returns the player's address either way; WriteTo sends to a player through the socket the
// player was last seen on (framed for the relay when that is the proxy socket).
type DualPacketConn struct {
	direct, proxy net.PacketConn
	in            chan dualPacket
	done          chan struct{}
	closeOnce     sync.Once

	mu         sync.Mutex
	relay      map[string]relayRoute // player address -> relay route
	lastDirect map[string]time.Time  // player address -> last seen directly
}

type relayRoute struct {
	relay  net.Addr // the relay's address as this socket sees it
	public *net.UDPAddr
	seen   time.Time
}

type dualPacket struct {
	b    []byte
	addr *net.UDPAddr
	err  error
}

// ListenGameUDP opens a game server's UDP socket. proxyAddress empty: a plain socket, as
// before. Otherwise a DualPacketConn with the proxy socket on proxyAddress.
func ListenGameUDP(address, proxyAddress string) (net.PacketConn, error) {
	direct, err := net.ListenPacket("udp", address)
	if err != nil || proxyAddress == "" {
		return direct, err
	}
	proxy, err := net.ListenPacket("udp", proxyAddress)
	if err != nil {
		_ = direct.Close()
		return nil, err
	}
	d := &DualPacketConn{direct: direct, proxy: proxy, in: make(chan dualPacket, 1024), done: make(chan struct{}),
		relay: map[string]relayRoute{}, lastDirect: map[string]time.Time{}}
	go d.pump(direct, false)
	go d.pump(proxy, true)
	go d.expire()
	return d, nil
}

func (d *DualPacketConn) pump(pc net.PacketConn, framed bool) {
	for {
		buf := make([]byte, 2048)
		n, from, err := pc.ReadFrom(buf)
		if err != nil {
			select {
			case d.in <- dualPacket{err: err}:
			case <-d.done:
			}
			return
		}
		b, addr := buf[:n], from.(*net.UDPAddr)
		if framed {
			h, rest, err := ParseProxyV2(b)
			if err != nil || h.Local {
				continue
			}
			player := &net.UDPAddr{IP: h.SrcIP, Port: h.SrcPort}
			public := &net.UDPAddr{IP: h.DstIP, Port: h.DstPort}
			d.mu.Lock()
			d.relay[player.String()] = relayRoute{relay: from, public: public, seen: time.Now()}
			d.mu.Unlock()
			b, addr = rest, player
		} else {
			d.mu.Lock()
			d.lastDirect[addr.String()] = time.Now()
			d.mu.Unlock()
		}
		select {
		case d.in <- dualPacket{b: b, addr: addr}:
		case <-d.done:
			return
		}
	}
}

// expire forgets routes not used for 10 minutes.
func (d *DualPacketConn) expire() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-d.done:
			return
		case now := <-t.C:
			d.mu.Lock()
			for k, r := range d.relay {
				if now.Sub(r.seen) > 10*time.Minute {
					delete(d.relay, k)
				}
			}
			for k, s := range d.lastDirect {
				if now.Sub(s) > 10*time.Minute {
					delete(d.lastDirect, k)
				}
			}
			d.mu.Unlock()
		}
	}
}

// ReadFrom implements net.PacketConn.
func (d *DualPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	select {
	case pk := <-d.in:
		if pk.err != nil {
			return 0, nil, pk.err
		}
		return copy(p, pk.b), pk.addr, nil
	case <-d.done:
		return 0, nil, net.ErrClosed
	}
}

// WriteTo implements net.PacketConn: through the relay for a player last seen through it,
// directly otherwise.
func (d *DualPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	ua, ok := addr.(*net.UDPAddr)
	if !ok {
		return d.direct.WriteTo(p, addr)
	}
	key := ua.String()
	d.mu.Lock()
	r, viaRelay := d.relay[key]
	lastDirect, seenDirect := d.lastDirect[key]
	d.mu.Unlock()
	if viaRelay && (!seenDirect || r.seen.After(lastDirect)) {
		frame := append(EncodeProxyV2(false, r.public.IP, r.public.Port, ua.IP, ua.Port), p...)
		if _, err := d.proxy.WriteTo(frame, r.relay); err != nil {
			return 0, err
		}
		return len(p), nil
	}
	return d.direct.WriteTo(p, addr)
}

// Close implements net.PacketConn.
func (d *DualPacketConn) Close() error {
	d.closeOnce.Do(func() { close(d.done) })
	e1, e2 := d.direct.Close(), d.proxy.Close()
	if e1 != nil {
		return e1
	}
	return e2
}

// LocalAddr implements net.PacketConn (the direct socket's).
func (d *DualPacketConn) LocalAddr() net.Addr { return d.direct.LocalAddr() }

// SetDeadline and friends apply to both sockets.
func (d *DualPacketConn) SetDeadline(t time.Time) error {
	_ = d.proxy.SetDeadline(t)
	return d.direct.SetDeadline(t)
}
func (d *DualPacketConn) SetReadDeadline(t time.Time) error {
	_ = d.proxy.SetReadDeadline(t)
	return d.direct.SetReadDeadline(t)
}
func (d *DualPacketConn) SetWriteDeadline(t time.Time) error {
	_ = d.proxy.SetWriteDeadline(t)
	return d.direct.SetWriteDeadline(t)
}
