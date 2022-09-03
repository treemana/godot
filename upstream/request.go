package upstream

import (
	"context"
	"net"

	"github.com/miekg/dns"

	"github.com/treemana/godot/log"
	"github.com/treemana/godot/util"
)

func (s *UpStream) request() {
	var resolversChan = make(chan *dns.Msg, len(s.resolvers))
	for dt := range s.dic {
		if len(dt.Request.Question) != 1 {
			log.Sugar.Warnf("sn=%d, id=%d, question=%d ", dt.SN, dt.Request.Id, len(dt.Request.Question))
			continue
		}

		if len(dt.Request.Answer) > 0 {
			log.Sugar.Warnf("sn=%d, id=%d, answer=%d", dt.SN, dt.Request.Id, len(dt.Request.Answer))
			continue
		}

		if dt.Response != nil {
			log.Sugar.Warnf("sn=%d, id=%d, response not nil", dt.SN, dt.Request.Id)
			continue
		}

		req := dt.Request.Copy()

		s.setSubnet(req, dt.RemoteAddr.IP)

		for index := range s.resolvers {
			go func(i int) {
				resolversChan <- resolve(context.TODO(), s.resolvers[i], req)
			}(index)
		}

		var answerMap = make(map[string]struct{})
		for n := len(s.resolvers); n > 0; n-- {
			response := <-resolversChan
			if dt.Response != nil || response == nil {
				continue
			}

			if response.Rcode != dns.RcodeSuccess {
				// something unusual happen
				log.Sugar.Warnf("sn=%d, id=%d, response code [%s]", dt.SN, dt.Request.Id, dns.RcodeToString[response.Rcode])
				dt.Response = response
				// do not break or return
				// the channel element filled by other resolvers need clean up
				continue
			}

			switch req.Question[0].Qtype {
			case dns.TypeA, dns.TypeAAAA:
				for _, rr := range response.Answer {
					ip := util.DNSSplitAnswer(rr)
					if len(ip) == 0 {
						continue
					}

					k := ip.String()
					if _, ok := answerMap[k]; ok {
						continue
					}

					answerMap[k] = struct{}{}
					dt.Answers = append(dt.Answers, rr)
				}
			default:
				// if query type is not A or AAAA, the first response by resolver will be return
				dt.Response = response
			}
		}

		// response should add to cache and s.doc
		if dt.Response != nil {
			s.doc <- dt
			continue
		}

		// query type A or AAAA need find the fastest one
		s.fastestChan <- dt
	}
}

// setSubnet set system subnet to dns.Msg EDNS0
// do nothing when req had a subnet already
func (s *UpStream) setSubnet(req *dns.Msg, ip net.IP) {
	if util.DNSSubnetExist(req) {
		return
	}

	var subnet *dns.EDNS0_SUBNET
	switch req.Question[0].Qtype {
	case dns.TypeA:
		subnet = s.subnetV4
	case dns.TypeAAAA:
		subnet = s.subnetV6
	default:
		if v4 := ip.To4(); v4 != nil {
			subnet = s.subnetV4
		} else {
			subnet = s.subnetV6
		}
	}

	util.DNSSetSUBNET(req, subnet)
}
