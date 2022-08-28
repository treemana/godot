package model

import (
	"net"

	"github.com/miekg/dns"
)

type DT struct {
	// SN serial number
	// 0 means cache update request, the response will not write to udp connection
	// other means the number of request from udp connection
	SN uint64 // serial number, 0

	// RemoteAddr the requester udp address
	RemoteAddr *net.UDPAddr

	Answers []dns.RR

	Request  *dns.Msg
	Response *dns.Msg

	Cached bool // when response from the cache, true will be set
}
