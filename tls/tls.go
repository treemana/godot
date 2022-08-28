package tls

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"net/url"
	"time"
)

const (
	timeoutDial      = time.Second
	timeoutHandshake = time.Second
)

// NewConn new a tls.Conn from url.URL
// return conn, elapse, error
func NewConn(ctx context.Context, u url.URL) (*tls.Conn, time.Duration, error) {

	ept := time.Now() // entry point time

	// dial
	dialer := &net.Dialer{Timeout: timeoutDial}
	start := time.Now()
	rawConn, err := dialer.DialContext(ctx, "tcp", u.Host)
	elapse := time.Since(start)
	if err != nil {
		return nil, math.MaxInt64, fmt.Errorf("dial [%+v], elapse %s", err, elapse)
	}

	// set deadline
	conn := tls.Client(rawConn, &tls.Config{ServerName: u.Hostname(), MinVersion: tls.VersionTLS13})
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
