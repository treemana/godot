package udp

import (
	"time"

	"github.com/miekg/dns"

	"github.com/treemana/godot/cache"
	"github.com/treemana/godot/log"
	"github.com/treemana/godot/util"
)

func (s *Server) write() {
	s.respWG.Add(1)
	for dt := range s.respChan {

		if dt.SN == 0 {
			log.Sugar.Warnf("sn=0 [%s]", dt.Request.Question[0].String())
			continue
		}

		if dt.Response == nil {
			log.Sugar.Errorf("sn=%d nil response", dt.SN)
			continue
		}

		if !util.DNSSubnetExist(dt.Request) {
			util.DNSSubnetRemove(dt.Response)
		}

		// update cache
		if !dt.Cached {
			cache.Update(dt.Response)
		}

		if dt.RemoteAddr == nil {
			log.Sugar.Debugf("sn=%d, remote addr nil, [%s]", dt.SN, dt.Request.Question[0].String())
			continue
		}

		bytes, err := dt.Response.Pack()
		if err != nil {
			log.Sugar.Warnf("sn=%d, response pack error=[%+v]", dt.SN, err)
			continue
		}

		if err = s.conn.SetWriteDeadline(time.Now().Add(defaultTimeout)); err != nil {
			log.Sugar.Fatalf("sn=%d, server udp connection set deadline error=[%+v]", dt.SN, err)
			continue
		}

		if _, err = s.conn.WriteToUDP(bytes, dt.RemoteAddr); err != nil {
			log.Sugar.Errorf("sn=%d, udp connection write error=[%+v]", dt.SN, err)
			// do not set break, s.respChan need be empty
			continue
		}

		// if _, _, err = s.conn.WriteMsgUDP(bytes, util.GetOOBWithSrc(s.address.IP), dt.RemoteAddr); err != nil {
		// 	log.Sugar.Errorf("sn=%d, udp connection write error=[%+v]", dt.SN, err)
		// 	// do not set break, s.respChan need be empty
		// 	continue
		// }
		log.Sugar.Infof("sn=%d, id=%d, cache=%t, %s answer %d", dt.SN, dt.Response.Id, dt.Cached, dns.RcodeToString[dt.Response.Rcode], len(dt.Response.Answer))
	}
	s.respWG.Done()
}
