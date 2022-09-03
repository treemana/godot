# Document

## DNS0 subnet mask length

The longer the prefix the higher its utility for user mapping but the greater the
privacy erosion. RFC 7871 [11.1. Privacy] recommends that recursive resolvers
truncate IPv4 addresses to at most 24 bits and IPv6 addresses to at
most 56 bits in the source prefix to maintain client privacy.

```text
Described in RFC 7871, Section 11.1.

https://github.com/miekg/dns/blob/v1.1.44/edns.go#L281-L317
https://github.com/AdguardTeam/dnsproxy/blob/master/proxyutil/udp_unix.go#L78-L87
https://github.com/AdguardTeam/dnsproxy/blob/master/upstream/bootstrap.go#L57-L78
https://dnsprivacy.org/public_resolvers/
```
