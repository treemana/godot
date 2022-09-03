package util

import (
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
	// ipify https://www.ipify.org/
	ipify4 = "https://api4.ipify.org/"
	ipify6 = "https://api6.ipify.org/"
)

var (
	// oobSize int

	pingNetwork = "tcp"
	pingPorts   = []string{"80", "443"}
	pingTimeout = time.Second
)

func init() {
	// oobSize = getOOBSize()
}

func GetPublicIPV4() (net.IP, error) { return getPublicIP(ipify4) }
func GetPublicIPV6() (net.IP, error) { return getPublicIP(ipify6) }
func getPublicIP(url string) (net.IP, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	var raw []byte
	if raw, err = io.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	return net.ParseIP(string(raw)), nil
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
// func getOOBSize() (oobSize int) {
// 	l4, l6 := len(ipv4.NewControlMessage(ipv4Flags)), len(ipv6.NewControlMessage(ipv6Flags))

// 	if l4 >= l6 {
// 		return l4
// 	}

// 	return l6
// }

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
