package main

import (
	"log"
	"net"
	"time"
)

func main() {
	log.SetFlags(log.Lmicroseconds)
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	log.Printf("SERVER listening for datagrams on %s (no accept/connection)", server.LocalAddr())
	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 64)
		for i := 0; i < 2; i++ {
			n, peer, err := server.ReadFromUDP(buf)
			if err != nil {
				done <- err
				return
			}
			log.Printf("SERVER datagram #%d bytes=%d from=%s payload=%q", i+1, n, peer, buf[:n])
			_, err = server.WriteToUDP(append([]byte("ACK:"), buf[:n]...), peer)
			if err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	client, err := net.DialUDP("udp", nil, server.LocalAddr().(*net.UDPAddr))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	for _, msg := range []string{"first", "second"} {
		log.Printf("CLIENT sends one datagram %q", msg)
		_, _ = client.Write([]byte(msg))
		n, err := client.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("CLIENT received one datagram %q", buf[:n])
	}
	if err := <-done; err != nil {
		log.Fatal(err)
	}
}
