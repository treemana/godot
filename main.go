package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/miekg/dns"

	"github.com/treemana/godot/log"
	"github.com/treemana/godot/udp"
	"github.com/treemana/godot/upstream"
	"github.com/treemana/godot/util"
)

// Option represents console arguments.  For further additions, please do not
// use the default option since it will cause some problems when config files
// are used.
type Option struct {
	Log struct {
		File    string `json:"file"`
		STDOUT  bool   `json:"stdout"`
		Verbose bool   `json:"verbose"`
	} `json:"log"`

	Server struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
	} `json:"server"`

	// CacheTTR cache time to refresh, number of minute
	// cache will be disabled if zero
	CacheTTR uint64 `json:"cache_ttr"`

	// upstream DNS resolvers
	Resolvers [][]string `json:"resolvers"`

	// ECS settings, ECS will disable when nil
	ECS *struct {
		IPV4       string `json:"ip_v4"`
		IPV6       string `json:"ip_v6"`
		MaskBitsV4 uint8  `json:"mask_bits_v4"`
		MaskBitsV6 uint8  `json:"mask_bits_v6"`
	} `json:"ecs"`
}

var (
	option Option
)

func main() {

	raw, err := os.ReadFile("godot.json")
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(raw, &option); err != nil {
		panic(err)
	}

	fmt.Println(string(raw))

	// init log
	if err = initLog(); err != nil {
		return
	}
	defer func() {
		_ = log.Logger.Sync()
		time.Sleep(time.Second)
	}()

	var server *udp.Server
	if server, err = InitServer(); err != nil {
		log.Sugar.Error(err)
		return
	}

	var subnets []*dns.EDNS0_SUBNET
	if subnets, err = getSubnets(); err != nil {
		log.Sugar.Error(err)
		return
	}

	var up *upstream.UpStream
	req, resp := server.GetChan()
	if up, err = upstream.New(option.Resolvers, subnets, req, resp); err != nil {
		log.Sugar.Error(err)
		return
	}

	up.Start()     // start upstream
	server.Start() // start server

	// godot is running until os exit
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	s := <-sc
	log.Sugar.Infof("signal %d %s", s, s)

	server.StopRead()
	up.Stop()
	server.StopWrite()
}

func initLog() error {
	lc := log.Config{
		File:       option.Log.File,
		STDOUT:     option.Log.STDOUT,
		MaxAge:     2,
		MaxSize:    10,
		MaxBackups: 100,
	}

	if option.Log.Verbose {
		lc.Level = -1
	}

	if err := log.Init(lc); err != nil {
		fmt.Println("log init error", err)
		return err
	}

	return nil
}

func InitServer() (*udp.Server, error) {
	ip := net.ParseIP(option.Server.Address)
	ttr := time.Minute * time.Duration(option.CacheTTR)
	return udp.New(ip, option.Server.Port, ttr)
}

func getSubnets() ([]*dns.EDNS0_SUBNET, error) {
	if option.ECS == nil {
		return nil, nil
	}

	var subnets = make([]*dns.EDNS0_SUBNET, 0, 2)

	subnetV4, err := getSubnet(option.ECS.IPV4, false, option.ECS.MaskBitsV4)
	if err != nil {
		return nil, err
	}
	subnets = append(subnets, subnetV4)

	var subnetV6 *dns.EDNS0_SUBNET
	if subnetV6, err = getSubnet(option.ECS.IPV6, true, option.ECS.MaskBitsV6); err != nil {
		return nil, err
	}
	subnets = append(subnets, subnetV6)

	return subnets, nil
}

func getSubnet(ipRAW string, v6 bool, mask uint8) (*dns.EDNS0_SUBNET, error) {

	var ip net.IP
	if len(ipRAW) == 0 {
		var err error
		if v6 {
			ip, err = util.GetPublicIPV6()
		} else {
			ip, err = util.GetPublicIPV4()
		}
		if err != nil {
			return nil, err
		}
	} else {
		ip = net.ParseIP(ipRAW)
	}

	if ip == nil {
		return nil, nil
	}

	if v6 {
		ip = ip.To16()
	} else {
		ip = ip.To4()
	}

	return util.DNSNewSubnetFromIP(ip, mask), nil
}
