package udp

import (
	"context"
	"time"

	"github.com/miekg/dns"

	"github.com/treemana/godot/cache"
	"github.com/treemana/godot/log"
	"github.com/treemana/godot/model"
)

func (s *Server) cacheFresher(ctx context.Context, ttr time.Duration) {

	if ttr <= 0 {
		return
	}

	cache.Start()

	var ticker = time.NewTicker(ttr)
	var i uint32

	for {
		select {
		case <-ticker.C:
			i++
			log.Sugar.Infof("server cache refresh %d start", i)
			s.reqWG.Add(1)
			for name, qTypes := range cache.GetAllQuestion() {
				if !s.status.Load() {
					log.Sugar.Info("server cache refresh after stopped")
					break
				}
				for _, qType := range qTypes {
					req := new(dns.Msg)
					req.SetQuestion(name, qType)
					s.reqChan <- &model.DT{SN: s.serial.Add(1), Request: req}
				}
			}
			s.reqWG.Done()
			log.Sugar.Infof("server cache refresh %d stop", i)
		case <-ctx.Done():
			log.Sugar.Info("server cache refresh stop")
			break
		}
	}
}
