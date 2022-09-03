package util

import (
	"fmt"
	"math"
	"net"
	"testing"
)

func Te1st1Ping(t *testing.T) {
	latency := Ping("cn.bing.com")
	fmt.Println(latency)
	fmt.Println(math.MaxUint32)
}

func TestPing(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		success bool
	}{
		{
			name:    "success",
			host:    "cn.bing.com",
			success: true,
		},
		{
			name:    "timeout",
			host:    "timeout.host",
			success: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Ping(tt.host); got < math.MaxUint32 != tt.success {
				t.Errorf("Ping() = %v, success %v", got, tt.success)
			}
		})
	}
}

func TestGetPublicIP(t *testing.T) {
	got, err := getPublicIP(ipify4)
	if err != nil {
		t.Errorf("getPublicIP() error = %v", err)
		return
	}
	if got == nil {
		t.Error("getPublicIP() got = nil")
	}

	fmt.Println(net.IPv6zero.String())
}
