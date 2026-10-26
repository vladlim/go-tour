package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func acceptKey(key string) string {
	s := sha1.Sum([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(s[:])
}

// writeFrame implements small, unfragmented RFC 6455 frames for this demo.
func writeFrame(w io.Writer, opcode byte, payload []byte, masked bool) error {
	if len(payload) > 125 {
		return fmt.Errorf("demo supports payloads up to 125 bytes")
	}
	header := []byte{0x80 | opcode, byte(len(payload))}
	if masked {
		header[1] |= 0x80
		mask := make([]byte, 4)
		if _, err := rand.Read(mask); err != nil {
			return err
		}
		header = append(header, mask...)
		copyPayload := append([]byte(nil), payload...)
		for i := range copyPayload {
			copyPayload[i] ^= mask[i%4]
		}
		payload = copyPayload
	}
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func readFrame(r io.Reader) (byte, []byte, bool, error) {
	h := make([]byte, 2)
	if _, err := io.ReadFull(r, h); err != nil {
		return 0, nil, false, err
	}
	masked, n := h[1]&0x80 != 0, int(h[1]&0x7f)
	if n == 126 || n == 127 {
		return 0, nil, false, fmt.Errorf("large frames omitted in demo")
	}
	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(r, mask); err != nil {
			return 0, nil, false, err
		}
	}
	p := make([]byte, n)
	if _, err := io.ReadFull(r, p); err != nil {
		return 0, nil, false, err
	}
	if masked {
		for i := range p {
			p[i] ^= mask[i%4]
		}
	}
	return h[0] & 0x0f, p, masked, nil
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("SERVER TCP listen %s", ln.Addr())
	done := make(chan error, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			done <- err
			return
		}
		defer c.Close()
		br := bufio.NewReader(c)
		req, err := http.ReadRequest(br)
		if err != nil {
			done <- err
			return
		}
		key := req.Header.Get("Sec-WebSocket-Key")
		log.Printf("SERVER HTTP Upgrade method=%s upgrade=%q connection=%q key=%q", req.Method, req.Header.Get("Upgrade"), req.Header.Get("Connection"), key)
		if !strings.EqualFold(req.Header.Get("Upgrade"), "websocket") || key == "" {
			done <- fmt.Errorf("invalid upgrade")
			return
		}
		accept := acceptKey(key)
		fmt.Fprintf(c, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", accept)
		log.Printf("SERVER sent 101; protocol switched from HTTP to WebSocket accept=%q", accept)
		op, p, masked, err := readFrame(br)
		if err != nil {
			done <- err
			return
		}
		log.Printf("SERVER frame opcode=%#x masked=%v payload=%q", op, masked, p)
		if !masked {
			done <- fmt.Errorf("client frame must be masked")
			return
		}
		if err = writeFrame(c, 0x1, []byte("echo:"+string(p)), false); err != nil {
			done <- err
			return
		}
		op, p, _, err = readFrame(br)
		if err != nil {
			done <- err
			return
		}
		log.Printf("SERVER control frame opcode=%#x payload=%q; responding pong", op, p)
		if err = writeFrame(c, 0xA, p, false); err != nil {
			done <- err
			return
		}
		op, p, _, err = readFrame(br)
		if err != nil {
			done <- err
			return
		}
		log.Printf("SERVER close frame opcode=%#x payload=%q; echoing close", op, p)
		done <- writeFrame(c, 0x8, p, false)
	}()

	c, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(2 * time.Second))
	br := bufio.NewReader(c)
	rawKey := make([]byte, 16)
	_, _ = rand.Read(rawKey)
	key := base64.StdEncoding.EncodeToString(rawKey)
	fmt.Fprintf(c, "GET /chat HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: %s\r\n\r\n", ln.Addr(), key)
	log.Printf("CLIENT sent HTTP Upgrade key=%q", key)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		log.Fatal(err)
	}
	expected := acceptKey(key)
	log.Printf("CLIENT status=%s accept-valid=%v", resp.Status, resp.Header.Get("Sec-WebSocket-Accept") == expected)
	log.Printf("CLIENT text frame (masked)")
	if err = writeFrame(c, 0x1, []byte("hello"), true); err != nil {
		log.Fatal(err)
	}
	op, p, masked, err := readFrame(br)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("CLIENT frame opcode=%#x masked=%v payload=%q", op, masked, p)
	log.Printf("CLIENT ping heartbeat")
	_ = writeFrame(c, 0x9, []byte("alive?"), true)
	op, p, _, err = readFrame(br)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("CLIENT pong opcode=%#x payload=%q", op, p)
	log.Printf("CLIENT graceful close")
	_ = writeFrame(c, 0x8, []byte{0x03, 0xE8}, true)
	op, _, _, err = readFrame(br)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("CLIENT close acknowledged opcode=%#x", op)
	if err = <-done; err != nil {
		log.Fatal(err)
	}
	log.Printf("RECONNECT: this connection is finished; recovery means a NEW TCP + HTTP Upgrade handshake, normally after exponential backoff with jitter and a resume token")
}
