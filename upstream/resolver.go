package upstream

import (
	"context"
	"net/url"
	"time"

	"github.com/miekg/dns"

	"github.com/treemana/godot/log"
	"github.com/treemana/godot/tls"
)

func resolve(ctx context.Context, u url.URL, req *dns.Msg) *dns.Msg {

	conn, _, err := tls.NewConn(ctx, u)
	if err != nil {
		return nil
	}
	defer func() { _ = conn.Close() }()

	var dnsConn = dns.Conn{Conn: conn}
	start := time.Now()
	if err = dnsConn.WriteMsg(req); err != nil {
		log.Sugar.Errorf("sending request to %s error=[%+v]", u.String(), err)
		return nil
	}

	var resp *dns.Msg
	if resp, err = dnsConn.ReadMsg(); err != nil {
		log.Sugar.Errorf("%s %s [%s]", u.String(), err, req.Question[0].String())
		return nil
	}
	elapsed := time.Since(start)

	if req.Id != resp.Id {
		log.Sugar.Info("unmatched request and response")
		return nil
	}

	log.Sugar.Debugf("%s response success, cost %s", u.String(), elapsed)

	return resp
}
