package udp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/treemana/godot/cache"
	"github.com/treemana/godot/log"
	"github.com/treemana/godot/model"
)

const (
	defaultTimeout = 10 * time.Second
)

type Configure struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type Server struct {
	address *net.UDPAddr
	conn    *net.UDPConn
	status  atomic.Bool // running status

	reqWG   sync.WaitGroup
	reqChan chan *model.DT // dns request

	respWG   sync.WaitGroup
	respChan chan *model.DT // dns response

	serial   atomic.Uint64
	cancelFn context.CancelFunc
}

func New(ip net.IP, port int, ttr time.Duration) (*Server, error) {

	if len(ip) == 0 {
		return nil, errors.New("invalid ip")
	}

	if port <= 0 {
		return nil, fmt.Errorf("invalid port=%d", port)
	}

	s := Server{
		address:  &net.UDPAddr{Port: port, IP: ip},
		reqChan:  make(chan *model.DT),
		respChan: make(chan *model.DT),
	}

	if err := s.setConn(); err != nil {
		return nil, fmt.Errorf("set conn error=[%+v]", err)
	}

	var ctx = context.TODO()
	ctx, s.cancelFn = context.WithCancel(ctx)
	go s.cacheFresher(ctx, ttr)

	return &s, nil
}

func (s *Server) GetChan() (chan *model.DT, chan *model.DT) {
	return s.reqChan, s.respChan
}

func (s *Server) Start() {

	s.status.Store(true)

	go s.read()
	go s.write()

	log.Sugar.Info("server running ...")

}

func (s *Server) StopRead() {
	log.Sugar.Info("server read stopping")
	s.status.Store(false)

	log.Sugar.Info("server waiting all request done")

	s.reqWG.Wait()
	log.Sugar.Info("server read stopped")

	close(s.reqChan)
	log.Sugar.Infof("server request chan closed, serial=%d", s.serial.Load())
}

func (s *Server) StopWrite() {

	log.Sugar.Info("server write stopping")

	cache.Stop()
	log.Sugar.Info("server cache stopped")

	close(s.respChan)
	log.Sugar.Info("server response chan closed")

	s.respWG.Wait()
	log.Sugar.Info("server write stopped")

	if err := s.conn.Close(); err != nil {
		log.Sugar.Errorf("server udp connection close error=[%+v]", err)
	}
}

func (s *Server) setConn() error {
	var err error
	if s.conn, err = net.ListenUDP("udp", s.address); err != nil {
		log.Sugar.Errorf("server udp [%s] listen error=[%+v]", s.address, err)
		return err
	}

	// if err = util.SetControlMessage(s.conn); err != nil {
	// 	defer func() { _ = s.conn.Close() }()
	// 	log.Sugar.Errorf("server udp [%s] connection set control error=[%+v]", s.address, err)
	// 	return err
	// }

	return nil
}
