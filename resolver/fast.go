package resolver

import (
	"context"
	"crypto/tls"
	"net/url"
	"time"

	"github.com/treemana/godot/log"
)

// GetFastFromURLs return the fastest(establish connection) *Resolver from raw urls string when the fastest exist
// or return nil
func GetFastFromURLs(rawURLs []string) *Resolver {
	var fast *Resolver
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

		r := NewResolver(u)
		if conn, elapse, err = r.getTLSConn(context.TODO()); err != nil {
			log.Sugar.Warnf("%s tls connection [%+v]", u.Host, err)
			continue
		}
		_ = conn.Close()

		if fast != nil && elapse >= min {
			continue
		}

		fast = r
		min = elapse
	}

	return fast
}

func GetFastFromURLGroups(groups [][]string) []*Resolver {

	if groups == nil {
		return make([]*Resolver, 0)
	}

	var hostMap = make(map[string]*Resolver)
	for _, group := range groups {
		r := GetFastFromURLs(group)
		if r == nil {
			continue
		}

		if _, ok := hostMap[r.u.Host]; ok {
			continue
		}

		hostMap[r.u.Host] = r
	}

	var resolvers = make([]*Resolver, 0, len(hostMap))
	var i int
	for _, r := range hostMap {
		resolvers = append(resolvers, r)
		log.Sugar.Infof("upstream resolver %d %s", i, r.u.Host)
		i++
	}

	return resolvers
}
