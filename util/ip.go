package util

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	wimiURLV4       = "https://api.whatismyip.com/wimi.php"
	wimiURLV6       = "https://apiv6.whatismyip.com/wimi.php"
	wimiHeaderKey   = "Origin"
	wimiHeaderValue = "https://www.whatismyip.com"
)

type WIMI struct {
	IP  string `json:"ip"` // string like "0.0.0.0"
	GEO string `json:"geo"`
	ISP string `json:"isp"`
}

var (
	oobSize int

	pingNetwork = "tcp"
	pingPorts   = []string{"80", "443"}
	pingTimeout = time.Second
)

func init() {
	oobSize = getOOBSize()
}

func GetPublicIPV4() (*WIMI, error) { return getPublicIP(false) }
func GetPublicIPV6() (*WIMI, error) { return getPublicIP(true) }
func getPublicIP(v6 bool) (*WIMI, error) {

	var url string
	if v6 {
		url = wimiURLV6
	} else {
		url = wimiURLV4
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(wimiHeaderKey, wimiHeaderValue)

	var res *http.Response
	if res, err = http.DefaultClient.Do(req); err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	var raw []byte
	if raw, err = io.ReadAll(res.Body); err != nil {
		return nil, err
	}

	var wimi WIMI
	if err = json.Unmarshal(raw, &wimi); err != nil {
		return nil, err
	}

	return &wimi, nil
}

func Read(c *net.UDPConn, buf []byte) (n int, remoteAddr *net.UDPAddr, err error) {
	// oob := make([]byte, oobSize)
	// n, _, _, remoteAddr, err = c.ReadMsgUDP(buf, oob)
	// if err != nil {
	// 	return -1, nil, err
	// }

	n, remoteAddr, err = c.ReadFromUDP(buf)
	if err != nil {
		return -1, nil, err
	}

	return n, remoteAddr, nil
}

func SetControlMessage(conn *net.UDPConn) error {

	err := ipv4.NewPacketConn(conn).SetControlMessage(ipv4Flags, true)
	if err != nil {
		fmt.Println("ipv4.NewPacketConn", err)
	} else {
		return nil
	}

	if err = ipv6.NewPacketConn(conn).SetControlMessage(ipv6Flags, true); err != nil {
		fmt.Println("ipv6.NewPacketConn", err)
		return err
	}

	return nil
}

// getOOBSize returns maximum size of the received OOB data.
func getOOBSize() (oobSize int) {
	l4, l6 := len(ipv4.NewControlMessage(ipv4Flags)), len(ipv6.NewControlMessage(ipv6Flags))

	if l4 >= l6 {
		return l4
	}

	return l6
}

// GetOOBWithSrc makes the OOB data with a specified source IP.
func GetOOBWithSrc(ip net.IP) []byte {
	if ip4 := ip.To4(); ip4 != nil {
		return (&ipv4.ControlMessage{Src: ip}).Marshal()
	}

	return (&ipv6.ControlMessage{Src: ip}).Marshal()
}

// Ping return the minimum latency in millisecond
// host : (net.IP).String()
// when dial error or timeout, return math.MaxUint32
func Ping(host string) uint32 {

	if len(host) == 0 {
		return math.MaxUint32
	}

	var c = make(chan uint32, len(pingPorts))
	defer close(c)
	for _, port := range pingPorts {
		addr := net.JoinHostPort(host, port)
		go ping(addr, c)
	}

	var min uint32 = math.MaxUint32
	for range pingPorts {
		latency := <-c
		if latency < min {
			min = latency
		}
	}

	return min
}

func ping(addr string, c chan uint32) string {
	var dialer = net.Dialer{Timeout: pingTimeout}

	start := time.Now()
	conn, err := dialer.Dial(pingNetwork, addr)
	if err != nil {
		c <- math.MaxUint32
		return fmt.Sprintf("dial %s error=[%+v]", addr, err)
	}
	elapsed := time.Since(start)

	defer func() {
		_ = conn.Close()
	}()

	c <- uint32(elapsed.Milliseconds())
	return ""
}
