package tls

import (
	"context"
	"crypto/tls"
	"net/url"
	"time"

	"github.com/treemana/godot/log"
)

// GetFastURL return the fastest(establish connection) *url.URL from raw url string when the fastest exist
// or return nil
func GetFastURL(rawURLs []string) *url.URL {
	var fast *url.URL
	var min, elapse time.Duration
	var conn *tls.Conn
	var hostMap = make(map[string]struct{}, len(rawURLs))

	for _, rawURL := range rawURLs {
		u, err := url.Parse(rawURL)
		if err != nil {
			log.Sugar.Warnf("%s parse error=[%+v]", rawURL, err)
			continue
		}

		if _, ok := hostMap[u.Host]; ok {
			continue
		}
		hostMap[u.Host] = struct{}{}

		if conn, elapse, err = NewConn(context.TODO(), *u); err != nil {
			log.Sugar.Warnf("%s tls connection [%+v]", u.Host, err)
			continue
		}
		_ = conn.Close()

		if fast == nil {
			fast = u
			min = elapse
			continue
		}

		if elapse > min {
			continue
		}
	}

	return fast
}

func GetFastURLs(groups [][]string) []url.URL {

	if groups == nil {
		return make([]url.URL, 0)
	}

	var hostMap = make(map[string]url.URL)
	for _, group := range groups {
		u := GetFastURL(group)
		if u == nil {
			continue
		}

		if _, ok := hostMap[u.Host]; ok {
			continue
		}

		hostMap[u.Host] = *u
	}

	var urls = make([]url.URL, 0, len(hostMap))
	for host := range hostMap {
		urls = append(urls, hostMap[host])
	}

	return urls
}
