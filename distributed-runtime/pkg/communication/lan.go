package communication

import (
	"fmt"
	"net"
	"strings"
	"time"
)

type LANInterfaceInfo struct {
	Name        string   `json:"name"`
	IPs         []string `json:"ips"`
	IsEthernet  bool     `json:"is_ethernet"`
	IsLoopback  bool     `json:"is_loopback"`
	PreferredIP string   `json:"preferred_ip"`
}

func DetectBestLANInterface() (*LANInterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}

	var bestEth *LANInterfaceInfo
	var bestWifi *LANInterfaceInfo
	var fallback *LANInterfaceInfo

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		validIPs := make([]string, 0)
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}

			ip4 := ipNet.IP.To4()
			if ip4 != nil && IsPrivateIP(ip4) {
				validIPs = append(validIPs, ip4.String())
			}
		}

		if len(validIPs) == 0 {
			continue
		}

		info := &LANInterfaceInfo{
			Name:        iface.Name,
			IPs:         validIPs,
			IsLoopback:  false,
			PreferredIP: validIPs[0],
		}

		lowerName := strings.ToLower(iface.Name)
		if strings.Contains(lowerName, "eth") || strings.Contains(lowerName, "ethernet") || strings.Contains(lowerName, "en") {
			info.IsEthernet = true
			if bestEth == nil {
				bestEth = info
			}
		} else if strings.Contains(lowerName, "wi-fi") || strings.Contains(lowerName, "wlan") || strings.Contains(lowerName, "wireless") {
			if bestWifi == nil {
				bestWifi = info
			}
		} else if fallback == nil {
			fallback = info
		}
	}

	if bestEth != nil {
		return bestEth, nil
	}
	if bestWifi != nil {
		return bestWifi, nil
	}
	if fallback != nil {
		return fallback, nil
	}

	return &LANInterfaceInfo{
		Name:        "loopback",
		IPs:         []string{"127.0.0.1"},
		IsLoopback:  true,
		PreferredIP: "127.0.0.1",
	}, nil
}

func IsPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	privateIPBlocks := []*net.IPNet{
		parseCIDR("10.0.0.0/8"),
		parseCIDR("172.16.0.0/12"),
		parseCIDR("192.168.0.0/16"),
		parseCIDR("127.0.0.0/8"),
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func parseCIDR(s string) *net.IPNet {
	_, block, _ := net.ParseCIDR(s)
	return block
}

func MeasureLANLatency(targetEndpoint string) (time.Duration, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", targetEndpoint, 2*time.Second)
	if err != nil {
		return 0, err
	}
	_ = conn.Close()
	return time.Since(start), nil
}
