package udp

import (
	"errors"
	"net"

	"github.com/miekg/dns"

	"github.com/treemana/godot/cache"
	"github.com/treemana/godot/log"
	"github.com/treemana/godot/model"
	"github.com/treemana/godot/util"
)

func (s *Server) produce(packet []byte, remote *net.UDPAddr, sn uint64) {

	var message = new(dns.Msg)
	if err := message.Unpack(packet); err != nil {
		log.Sugar.Errorf("sn=%d server unpack error=[%+v], raw=[%s]", sn, err, packet)
		return
	}

	if len(message.Answer) > 0 {
		log.Sugar.Warnf("sn=%d, id=%d already answered", sn, message.MsgHdr.Id)
		return
	}

	dt := &model.DT{
		SN:         sn,
		Request:    message,
		RemoteAddr: remote,
	}

	log.Sugar.Infof("sn=%d, id=%d, query=[%s]", sn, message.MsgHdr.Id, message.Question[0].String())

	// local cache hit
	if dt.Response = cache.Get(dt.Request); dt.Response != nil {
		dt.Cached = true
		s.respChan <- dt
		return
	}

	s.reqChan <- dt
}

func (s *Server) read() {
	bytes := make([]byte, dns.DefaultMsgSize)
	for {
		n, remoteAddr, err := util.Read(s.conn, bytes)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Sugar.Warn("server read connection closed")
				break
			}
			log.Sugar.Error("server read error : ", err)
			continue
		}

		if n <= 0 {
			log.Sugar.Warn("server read 0 byte")
			continue
		}

		s.reqWG.Add(1)

		if !s.status.Load() {
			log.Sugar.Info("server read after stopped")
			break
		}

		// documentation says to handle the packet even if err occurs, so do that first
		// make a copy of all bytes because ReadFrom() will overwrite contents of b on next call
		// we need the contents to survive the call because we're handling them in goroutine
		packet := make([]byte, n)
		copy(packet, bytes)

		go func() {
			s.produce(packet, remoteAddr, s.serial.Add(1))
			s.reqWG.Done()
		}()
	}
}
