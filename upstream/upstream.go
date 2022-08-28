package upstream

import (
	"errors"
	"net/url"
	"sync"

	"github.com/miekg/dns"

	"github.com/treemana/godot/log"
	"github.com/treemana/godot/model"
	"github.com/treemana/godot/tls"
)

const (
	ttl uint32 = 3600 // one hour equals 3600 seconds
)

type UpStream struct {
	subnetV4  *dns.EDNS0_SUBNET
	subnetV6  *dns.EDNS0_SUBNET
	resolvers []url.URL

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
		resolvers: tls.GetFastURLs(rawURLGroups),
		dic:       reqChan,
		doc:       respChan,
		respNum:   2,
	}

	if len(us.resolvers) == 0 {
		return nil, errors.New("empty UpStreams")
	}

	for i, resolver := range us.resolvers {
		log.Sugar.Infof("upstream resolver %d %s", i, resolver.Host)
	}

	for _, subnet := range subnets {
		if subnet == nil {
			continue
		}

		log.Sugar.Infof("upstream subnet %s", subnet.String())

		if v4 := subnet.Address.To4(); v4 != nil {
			us.subnetV4 = subnet
			continue
		}

		us.subnetV6 = subnet
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
