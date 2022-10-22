package upstream

import (
	"errors"
	"sync"

	"github.com/miekg/dns"

	"github.com/treemana/godot/log"
	"github.com/treemana/godot/model"
	"github.com/treemana/godot/resolver"
)

const (
	ttl uint32 = 3600 // one hour equals 3600 seconds
)

type UpStream struct {
	subnetV4  *dns.EDNS0_SUBNET
	subnetV6  *dns.EDNS0_SUBNET
	resolvers []*resolver.Resolver

	// dt in/out channel
	dic chan *model.DT
	doc chan *model.DT

	reqWG       sync.WaitGroup
	respNum     int
	respWG      sync.WaitGroup
	fastestChan chan *model.DT
}

func New(rawURLGroups [][]string, subnets []*dns.EDNS0_SUBNET, reqChan, respChan chan *model.DT) (*UpStream, error) {
	if len(rawURLGroups) == 0 {
		return nil, errors.New("empty rawURLGroups")
	}

	us := &UpStream{
		resolvers: resolver.GetFastFromURLGroups(rawURLGroups),
		dic:       reqChan,
		doc:       respChan,
		respNum:   2,
	}

	if len(us.resolvers) == 0 {
		return nil, errors.New("empty UpStreams")
	}

	for _, subnet := range subnets {
		if subnet == nil {
			continue
		}

		log.Sugar.Infof("upstream subnet %s", subnet.String())

		if v6 := subnet.Address.To16(); v6 != nil {
			us.subnetV6 = subnet
			continue
		}
		us.subnetV4 = subnet
	}

	return us, nil
}

func (s *UpStream) Start() {
	s.fastestChan = make(chan *model.DT, s.respNum)
	s.respWG.Add(s.respNum)
	for i := 0; i < s.respNum; i++ {
		go func() {
			s.response()
			s.respWG.Done()
		}()
	}

	s.reqWG.Add(1)
	go func() {
		s.request()
		s.reqWG.Done()
	}()
	log.Sugar.Info("upstream is running ...")
}

func (s *UpStream) Stop() {
	log.Sugar.Info("upstream stopping")
	s.reqWG.Wait()
	close(s.fastestChan)
	log.Sugar.Info("upstream reply chan closed")
	s.respWG.Wait()
	log.Sugar.Info("upstream stopped")
}
