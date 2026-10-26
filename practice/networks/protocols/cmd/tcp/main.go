package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

func main() {
	log.SetFlags(log.Lmicroseconds)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("SERVER listen %s", ln.Addr())
	done := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		log.Printf("SERVER accepted local=%s remote=%s", conn.LocalAddr(), conn.RemoteAddr())
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		r := bufio.NewReader(conn)
		for {
			line, err := r.ReadString('\n')
			if err == io.EOF {
				log.Printf("SERVER got EOF: client half-closed write side")
				done <- nil
				return
			}
			if err != nil {
				done <- err
				return
			}
			log.Printf("SERVER framed message=%q", line)
			fmt.Fprintf(conn, "ACK %s", line)
		}
	}()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		log.Fatal(err)
	}
	tcp := conn.(*net.TCPConn)
	log.Printf("CLIENT connected local=%s remote=%s", conn.LocalAddr(), conn.RemoteAddr())
	for _, msg := range []string{"one\n", "two\n"} {
		log.Printf("CLIENT write %q", msg)
		_, _ = io.WriteString(conn, msg)
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("CLIENT response=%q", line)
	}
	log.Printf("CLIENT CloseWrite (FIN; reading would still be possible)")
	_ = tcp.CloseWrite()
	if err := <-done; err != nil {
		log.Fatal(err)
	}
	_ = conn.Close()
}
