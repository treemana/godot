package upstream

import (
	"math"
	"sync"

	"github.com/miekg/dns"

	"github.com/treemana/godot/log"
	"github.com/treemana/godot/util"
)

// response handle query A or AAAA answers
// return the fastest answer when the fastest answer exist
// return DNS failed reply when all answers can't establish connection
func (s *UpStream) response() {
	for dt := range s.fastestChan {
		log.Sugar.Infof("sn=%d, id=%d, len(dt.Answers)=%d", dt.SN, dt.Request.MsgHdr.Id, len(dt.Answers))

		if len(dt.Answers) == 0 {
			dt.Response = util.DNSNewNXDomain(dt.Request)
			s.doc <- dt
			continue
		}

		var latencies = make([]uint32, len(dt.Answers))

		var wg sync.WaitGroup
		wg.Add(len(dt.Answers))
		for i := range dt.Answers {
			go func(index int) {
				latencies[index] = util.Ping(util.DNSSplitAnswer(dt.Answers[index]).String())
				wg.Done()
			}(i)
		}
		wg.Wait()

		var fastest int
		for i, latency := range latencies {
			if latency < latencies[fastest] {
				fastest = i
			}
		}

		if latencies[fastest] == math.MaxUint32 {
			dt.Response = util.DNSNewNXDomain(dt.Request)
			s.doc <- dt
			continue
		}

		dt.Answers[fastest].Header().Ttl = ttl

		dt.Response = util.DNSNewResponseByAnswer(dt.Request, []dns.RR{dt.Answers[fastest]})

		// resolve
		s.doc <- dt
	}
}
