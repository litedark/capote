package main

import (
	"flag"
	"log"
	"net"
	"os"
)

var dnsaddr = flag.String("dns", "0.0.0.0:5353", "DNS listen address")
var httpaddr = flag.String("http", "0.0.0.0:10080", "HTTP proxy listen address")
var httpsaddr = flag.String("https", "0.0.0.0:10443", "HTTPS proxy listen address")
var portalfile = flag.String("portal", "", "Captive portal html filename")
var socksaddr = flag.String("socks", "127.0.0.1:9050", "SOCKS service address")

var locals = make(map[*net.IPNet]net.IP)

var portal string

func main() {
	if !flag.Parsed() {
		flag.Parse()
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}
	for _, a := range addrs {
		ip, ipnet, err := net.ParseCIDR(a.String())
		if err != nil {
			log.Fatal(err)
		}
		locals[ipnet] = ip
	}
	if portal = *portalfile; len(portal) > 0 {
		fi, err := os.Stat(portal)
		if err != nil {
			log.Fatal(err)
		}
		if !fi.Mode().IsRegular() {
			log.Fatalf("%s is not a regular file", portal)
		}
	}
	errchan := make(chan error, 1)
	dnshandler := new(DNSHandler)
	go prox(errchan)
	go ListenAndServe(*dnsaddr, "udp4", dnshandler, errchan)
	go ListenAndServe(*dnsaddr, "tcp4", dnshandler, errchan)
	log.Fatal(<-errchan)
}
