package main

import (
	"context"
	"log"
	"net"
	"time"
)

func main() {
	log.SetFlags(log.Lmicroseconds)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r := net.DefaultResolver
	for _, host := range []string{"localhost", "example.com"} {
		log.Printf("RESOLVER lookup A/AAAA for %q", host)
		ips, err := r.LookupIPAddr(ctx, host)
		if err != nil {
			log.Printf("lookup failed (network may be offline): %v", err)
			continue
		}
		for _, ip := range ips {
			log.Printf("ANSWER host=%s ip=%s ipv4=%v", host, ip.IP, ip.IP.To4() != nil)
		}
	}
	log.Printf("RESOLVER reverse/PTR lookup for 127.0.0.1")
	names, err := r.LookupAddr(ctx, "127.0.0.1")
	if err != nil {
		log.Printf("reverse lookup failed: %v", err)
		return
	}
	log.Printf("PTR answers=%v", names)
}
