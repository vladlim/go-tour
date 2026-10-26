package main

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"roadmap/networks/internal/testcert"
	"time"
)

func main() {
	log.SetFlags(log.Lmicroseconds)
	cert, roots, err := testcert.New()
	if err != nil {
		log.Fatal(err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS13})
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("SERVER TCP listen with TLS 1.3 on %s", ln.Addr())
	done := make(chan error, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			done <- err
			return
		}
		defer c.Close()
		t := c.(*tls.Conn)
		log.Printf("SERVER accepted TCP; starting TLS handshake")
		if err := t.Handshake(); err != nil {
			done <- err
			return
		}
		s := t.ConnectionState()
		log.Printf("SERVER handshake complete version=%#x cipher=%s SNI=%q", s.Version, tls.CipherSuiteName(s.CipherSuite), s.ServerName)
		line, err := bufio.NewReader(t).ReadString('\n')
		if err == nil {
			log.Printf("SERVER decrypted application data=%q", line)
			_, err = t.Write([]byte("secure pong\n"))
		}
		done <- err
	}()

	d := net.Dialer{Timeout: time.Second}
	c, err := tls.DialWithDialer(&d, "tcp", ln.Addr().String(), &tls.Config{RootCAs: roots, ServerName: "localhost", MinVersion: tls.VersionTLS13})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	s := c.ConnectionState()
	log.Printf("CLIENT verified certificate=%q; version=%#x cipher=%s", s.PeerCertificates[0].Subject.CommonName, s.Version, tls.CipherSuiteName(s.CipherSuite))
	_, _ = c.Write([]byte("secure ping\n"))
	line, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("CLIENT decrypted response=%q", line)
	if err := <-done; err != nil {
		log.Fatal(err)
	}
}
