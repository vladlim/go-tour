package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	log.SetFlags(log.Lmicroseconds)
	ip := net.ParseIP("192.168.1.10")
	_, subnet, _ := net.ParseCIDR("192.168.1.10/24")
	log.Printf("parsed IP=%s; subnet=%s; contains 192.168.1.42=%v", ip, subnet, subnet.Contains(net.ParseIP("192.168.1.42")))

	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}
	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		log.Printf("interface=%q mtu=%d flags=%s addresses=%v", iface.Name, iface.MTU, iface.Flags, addrs)
	}

	host, port, _ := net.SplitHostPort("[::1]:8080")
	fmt.Printf("IPv6 endpoint split safely: host=%q port=%q\n", host, port)
}
