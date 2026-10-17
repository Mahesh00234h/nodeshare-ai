package network

import (
	"net"
	"strings"
)

type NetworkInterface struct {
	Name               string   `json:"name"`
	IPs                []string `json:"ips"`
	MAC                string   `json:"mac,omitempty"`
	IsWired            bool     `json:"is_wired"`
	EstimatedBandwidth uint64   `json:"estimated_bandwidth_bps"` // e.g. 1 Gbps
}

type NetworkInfo struct {
	Interfaces         []NetworkInterface `json:"interfaces"`
	EstimatedBandwidth uint64             `json:"estimated_bandwidth"`
}

func Detect() (*NetworkInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return &NetworkInfo{}, err
	}

	res := &NetworkInfo{
		Interfaces: make([]NetworkInterface, 0),
	}

	for _, iface := range ifaces {
		// Ignore loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		ipStrs := make([]string, 0)
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil || ipNet.IP.To16() != nil {
					ipStrs = append(ipStrs, ipNet.IP.String())
				}
			}
		}

		if len(ipStrs) == 0 {
			continue
		}

		nameLower := strings.ToLower(iface.Name)
		isWired := !strings.Contains(nameLower, "wlan") && !strings.Contains(nameLower, "wi-fi") && !strings.Contains(nameLower, "wireless")

		netIface := NetworkInterface{
			Name:               iface.Name,
			IPs:                ipStrs,
			MAC:                iface.HardwareAddr.String(),
			IsWired:            isWired,
			EstimatedBandwidth: 1000000000, // 1 Gbps default estimate
		}

		res.Interfaces = append(res.Interfaces, netIface)
	}

	if len(res.Interfaces) > 0 {
		res.EstimatedBandwidth = 1000000000
	}

	return res, nil
}
