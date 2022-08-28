package util

import (
	"net"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	IPV4MaskBitsMax     = net.IPv4len * 8
	IPV4MaskBitsDefault = 24 // RFC 7871 Section 11.1
	IPV6MaskBitsMax     = net.IPv6len * 8
	IPV6MaskBitsDefault = 56 // RFC 7871 Section 11.1

	// ipv*Flags is the set of socket option flags for configuring IPv* UDP
	// connection to receive an appropriate OOB data.  For both versions the flags
	// are:
	//   FlagDst
	//   FlagInterface
	ipv4Flags = ipv4.FlagDst | ipv4.FlagInterface
	ipv6Flags = ipv6.FlagDst | ipv6.FlagInterface
)

func DNSNewSubnetFromIP(ip net.IP, maskBits uint8) *dns.EDNS0_SUBNET {

	if ip == nil || len(ip) == 0 {
		return nil
	}
	// A Stub Resolver MUST set SCOPE PREFIX-LENGTH to 0. See RFC 7871 Section 6.

	var subnet = &dns.EDNS0_SUBNET{
		Code: dns.EDNS0SUBNET,
	}

	if ip4 := ip.To4(); ip4 != nil {
		if maskBits > IPV4MaskBitsMax {
			subnet.SourceNetmask = IPV4MaskBitsMax
		} else if maskBits <= 0 {
			subnet.SourceNetmask = IPV4MaskBitsDefault
		} else {
			subnet.SourceNetmask = maskBits
		}
		subnet.Family = 1
		subnet.Address = ip4
		return subnet
	}

	// ipv6
	if maskBits > IPV6MaskBitsMax {
		subnet.SourceNetmask = IPV6MaskBitsMax
	} else if maskBits <= 0 {
		subnet.SourceNetmask = IPV6MaskBitsDefault
	} else {
		subnet.SourceNetmask = maskBits
	}
	subnet.Family = 2
	subnet.Address = ip

	return subnet
}

func DNSNewFailure(source *dns.Msg) *dns.Msg {
	if source == nil {
		return nil
	}

	var target = new(dns.Msg)
	target.SetRcode(source, dns.RcodeServerFailure)
	target.RecursionAvailable = true

	return target
}

func DNSSplitAnswer(rr dns.RR) net.IP {
	switch rr := rr.(type) {
	case *dns.A:
		return rr.A.To4()
	case *dns.AAAA:
		return rr.AAAA
	default:
		return nil
	}
}

// DNSSetSUBNET set the EDNS client subnet option
// return true only when the subnet added to m
func DNSSetSUBNET(m *dns.Msg, subnet *dns.EDNS0_SUBNET) {

	if m == nil || subnet == nil {
		return
	}

	var opt = m.IsEdns0()

	if opt == nil {
		opt = &dns.OPT{
			Hdr:    dns.RR_Header{Name: ".", Rrtype: dns.TypeOPT},
			Option: []dns.EDNS0{subnet},
		}
		opt.SetUDPSize(dns.DefaultMsgSize)
		m.Extra = append(m.Extra, opt)
		return
	}

	if len(opt.Option) > 0 {
		for _, edns0 := range opt.Option {
			if edns0.Option() == dns.EDNS0SUBNET {
				// subnet already exist
				return
			}
		}
	}

	opt.Option = append(opt.Option, subnet)
}

func DNSNewResponseByAnswer(req *dns.Msg, answer []dns.RR) *dns.Msg {
	if req == nil || len(answer) == 0 {
		return nil
	}

	var resp = new(dns.Msg)
	resp.SetReply(req)
	resp.RecursionAvailable = true
	resp.AuthenticatedData = false
	resp.Answer = answer

	return resp
}

// DNSSubnetRemove set the EDNS client subnet option
// return true only when the subnet added to m
func DNSSubnetRemove(m *dns.Msg) {

	if m == nil {
		return
	}

	var opt = m.IsEdns0()
	if opt == nil || len(opt.Option) == 0 {
		return
	}

	var index = -1
	for i, edns0 := range opt.Option {
		if edns0.Option() == dns.EDNS0SUBNET {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	var options = make([]dns.EDNS0, 0, len(opt.Option)-1)
	options = append(opt.Option[:index], opt.Option[index+1:]...)
	opt.Option = options
	return
}

func DNSSubnetExist(m *dns.Msg) bool {

	if m == nil {
		return false
	}

	var opt = m.IsEdns0()
	if opt == nil || len(opt.Option) == 0 {
		return false
	}

	for _, edns0 := range opt.Option {
		if edns0.Option() == dns.EDNS0SUBNET {
			return true
		}
	}

	return false
}
