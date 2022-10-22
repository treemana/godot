package resolver

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"net/url"
	"time"

	"github.com/miekg/dns"

	"github.com/treemana/godot/log"
)

const (
	timeoutDial      = time.Second
	timeoutHandshake = time.Second
)

type Resolver struct {
	u      *url.URL
	config *tls.Config
}

func NewResolver(u *url.URL) *Resolver {
	return &Resolver{
		u:      u,
		config: &tls.Config{ServerName: u.Hostname(), MinVersion: tls.VersionTLS13, ClientSessionCache: tls.NewLRUClientSessionCache(0)},
	}
}

func (r *Resolver) Resolve(ctx context.Context, req *dns.Msg) *dns.Msg {
	conn, _, err := r.getTLSConn(ctx)
	if err != nil {
		return nil
	}
	defer func() { _ = conn.Close() }()

	fmt.Println(r.u.Hostname(), conn.ConnectionState().DidResume)

	var dnsConn = dns.Conn{Conn: conn}
	start := time.Now()
	if err = dnsConn.WriteMsg(req); err != nil {
		log.Sugar.Errorf("sending request to %s error=[%+v]", r.u.String(), err)
		return nil
	}

	var resp *dns.Msg
	if resp, err = dnsConn.ReadMsg(); err != nil {
		log.Sugar.Errorf("%s %s [%s]", r.u.String(), err, req.Question[0].String())
		return nil
	}
	elapsed := time.Since(start)

	if req.Id != resp.Id {
		log.Sugar.Info("unmatched request and response")
		return nil
	}

	log.Sugar.Debugf("%s response success, cost %s", r.u.String(), elapsed)

	return resp
}

func (r *Resolver) getTLSConn(ctx context.Context) (*tls.Conn, time.Duration, error) {
	ept := time.Now() // entry point time

	// dial
	dialer := &net.Dialer{Timeout: timeoutDial}
	start := time.Now()
	rawConn, err := dialer.DialContext(ctx, "tcp", r.u.Host)
	elapse := time.Since(start)
	if err != nil {
		return nil, math.MaxInt64, fmt.Errorf("dial [%+v], elapse %s", err, elapse)
	}

	// set deadline
	conn := tls.Client(rawConn, r.config)
	start = time.Now()
	err = conn.SetDeadline(time.Now().Add(timeoutHandshake))
	elapse = time.Since(start)
	if err != nil {
		_ = conn.Close()
		return nil, math.MaxInt64, fmt.Errorf("set deadline [%+v], elapse %s", err, elapse)
	}

	// handshake
	start = time.Now()
	err = conn.Handshake()
	elapse = time.Since(start)
	if err != nil {
		_ = conn.Close()
		return nil, math.MaxInt64, fmt.Errorf("handshake [%+v], elapse %s", err, elapse)
	}

	return conn, time.Since(ept), nil
}
