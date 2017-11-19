package main

import (
	"net"
	"strings"

	"github.com/miekg/dns"
)

type DNSHandler struct{}

var proxied = dns.A{Hdr: dns.RR_Header{Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}, A: net.ParseIP("10.0.0.1")}

func (h *DNSHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.MsgHdr.RecursionAvailable = true
	defer w.WriteMsg(m)
	if len(r.Question) != 1 {
		m.SetRcode(r, dns.RcodeRefused)
		return
	}
	q := r.Question[0]
	if q.Qclass != dns.ClassINET {
		m.SetRcode(r, dns.RcodeRefused)
		return
	}
	if !strings.HasSuffix(q.Name, ".onion.") {
		m.SetRcode(r, dns.RcodeRefused)
		return
	}
	switch q.Qtype {
	case dns.TypeA, dns.TypeANY:
		addr, _, err := net.SplitHostPort(w.LocalAddr().String())
		if err != nil {
			m.SetRcode(r, dns.RcodeServerFailure)
			return
		}
		ip := net.ParseIP(addr)
		if ip != nil && ip.IsGlobalUnicast() {
			m.SetRcode(r, dns.RcodeSuccess)
			a := proxied
			a.A = ip
			a.Hdr.Name = q.Name
			m.Answer = []dns.RR{&a}
			return
		}
		addr, _, err = net.SplitHostPort(w.RemoteAddr().String())
		ip = net.ParseIP(addr)
		if ip == nil {
			m.SetRcode(r, dns.RcodeServerFailure)
			return
		}
		for ipnet, localip := range locals {
			if ipnet.Contains(ip) {
				m.SetRcode(r, dns.RcodeSuccess)
				a := proxied
				a.A = localip
				a.Hdr.Name = q.Name
				m.Answer = []dns.RR{&a}
				return
			}
		}
		m.SetRcode(r, dns.RcodeServerFailure)
		return
	default:
		m.SetRcode(r, dns.RcodeNameError)
		return
	}
}

func ListenAndServe(addr, network string, handler dns.Handler, errchan chan<- error) {
	errchan <- dns.ListenAndServe(addr, network, handler)
}
