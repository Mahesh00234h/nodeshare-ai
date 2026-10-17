package communication

import (
	"fmt"
	"net"
	"strings"
)

type LANSecurityGuard struct {
	AllowedCIDRs []*net.IPNet
}

func NewLANSecurityGuard(customAllowedCIDRs []string) *LANSecurityGuard {
	guard := &LANSecurityGuard{
		AllowedCIDRs: make([]*net.IPNet, 0),
	}

	defaultCIDRs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}

	for _, cidr := range append(defaultCIDRs, customAllowedCIDRs...) {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			guard.AllowedCIDRs = append(guard.AllowedCIDRs, block)
		}
	}
	return guard
}

func (sg *LANSecurityGuard) ValidateRPCEndpoint(endpoint string) error {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		host = endpoint
	}

	ip := net.ParseIP(host)
	if ip == nil {
		addrs, err := net.LookupIP(host)
		if err != nil || len(addrs) == 0 {
			return fmt.Errorf("SECURITY_VIOLATION: cannot resolve RPC endpoint host %s: %v", host, err)
		}
		ip = addrs[0]
	}

	for _, block := range sg.AllowedCIDRs {
		if block.Contains(ip) {
			return nil
		}
	}

	return fmt.Errorf("SECURITY_VIOLATION: RPC endpoint %s (%s) is not on an approved local LAN interface", endpoint, ip.String())
}

func (sg *LANSecurityGuard) ValidateBindHost(bindHost string) error {
	if bindHost == "0.0.0.0" || bindHost == "" {
		return nil
	}

	ip := net.ParseIP(bindHost)
	if ip == nil {
		return fmt.Errorf("SECURITY_VIOLATION: invalid bind host IP %s", bindHost)
	}

	for _, block := range sg.AllowedCIDRs {
		if block.Contains(ip) {
			return nil
		}
	}

	return fmt.Errorf("SECURITY_VIOLATION: RPC bind host %s is a public/non-LAN interface", bindHost)
}

func IsPublicHost(host string) bool {
	if strings.Contains(host, "render.com") || strings.Contains(host, "onrender.com") {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil && !IsPrivateIP(ip) {
		return true
	}
	return false
}
