package cache

/*

benchmark on macOS 12.4(intel) with go 1.18

read speed
  sync.RWMutex : atomic.LoadPointer = 1 : 9

write speed
  sync.RWMutex : atomic.StorePointer = 6 : 1

*/

import (
	"sync"
	"sync/atomic"

	"github.com/miekg/dns"

	"github.com/treemana/godot/log"
	"github.com/treemana/godot/util"
)

type reply map[string]map[uint16]any

var (
	rm     atomic.Pointer[reply]
	enable atomic.Bool

	wg sync.WaitGroup
	uc chan *dns.Msg
)

func Start() {
	rm.Store(&reply{})
	uc = make(chan *dns.Msg)
	go update()
	enable.Store(true)
}

func Stop() {
	if !enable.Load() {
		return
	}

	log.Sugar.Info("cache stopping")
	enable.Store(false)
	log.Sugar.Info("cache waiting")
	wg.Wait()
	close(uc)
	log.Sugar.Info("cache stopped")
}

func Get(request *dns.Msg) *dns.Msg {
	if request == nil || !enable.Load() {
		return nil
	}

	var q = request.Question[0]
	var m = *rm.Load()
	if len(m) == 0 || len(m[q.Name]) == 0 || m[q.Name][q.Qtype] == nil {
		return nil
	}

	switch request.Question[0].Qtype {
	case dns.TypeA, dns.TypeAAAA:
		return util.DNSNewResponseByAnswer(request, m[q.Name][q.Qtype].([]dns.RR))
	default:
		response := (m[q.Name][q.Qtype].(*dns.Msg)).Copy()
		response.Id = request.Id
		return response
	}
}

// GetAllQuestion return all cached request questions
// map[question name][]uint16{question type}
func GetAllQuestion() map[string][]uint16 {
	var origin = *rm.Load()
	var all = make(map[string][]uint16, len(origin))
	for name, typeMap := range origin {
		if len(name) == 0 || len(typeMap) == 0 {
			continue
		}
		var types = make([]uint16, 0, len(typeMap))
		for qType := range typeMap {
			types = append(types, qType)
		}
		all[name] = types
	}
	return all
}

func Update(response *dns.Msg) {

	if response == nil || response.Rcode != dns.RcodeSuccess || len(response.Answer) == 0 || !enable.Load() {
		return
	}

	wg.Add(1)
	if !enable.Load() {
		log.Sugar.Info("cache update after stopped")
		wg.Done()
		return
	}

	go func() {
		uc <- response
		wg.Done()
	}()

}

func update() {
	var q dns.Question
	var m reply
	for message := range uc {
		q = message.Question[0]
		m = *rm.Load()
		target := duplicate(m, q.Name, q.Qtype)
		switch q.Qtype {
		case dns.TypeA, dns.TypeAAAA:
			target[q.Name][q.Qtype] = message.Answer
		default:
			target[q.Name][q.Qtype] = message
		}
		rm.Store(&target)
	}
}

func duplicate(source reply, host string, qType uint16) reply {

	if len(source) == 0 {
		return reply{host: {qType: nil}}
	}

	var (
		target reply
		hb     bool // host bit, true when host exist in source
		tb     bool // qType bit, true when (hb id true and qType exist in source)
	)

	if _, hb = source[host]; hb {
		_, tb = source[host][qType]
		target = make(map[string]map[uint16]any, len(source))
	} else {
		target = make(map[string]map[uint16]any, len(source)+1)
		target[host] = map[uint16]any{qType: nil}
	}

	for k, vSource := range source {
		var vTarget map[uint16]any

		if k == host && hb && !tb {
			vTarget = make(map[uint16]any, len(vSource)+1)
		} else {
			vTarget = make(map[uint16]any, len(vSource))
		}

		for u, a := range vSource {
			vTarget[u] = a
		}

		target[k] = vTarget
	}

	return target
}
