package protocol

import (
	protov1 "github.com/distributed-ai/node-agent/api/proto/v1"
	"github.com/distributed-ai/node-agent/internal/hardware"
)

const ProtocolVersion = "v1"

// ToRegistrationRequest converts local hardware report and node credentials into a RegistrationRequest proto.
func ToRegistrationRequest(report *hardware.FullHardwareReport, nodeID string, authToken string) *protov1.RegistrationRequest {
	req := &protov1.RegistrationRequest{
		NodeId:              nodeID,
		AuthToken:           authToken,
		Hostname:            report.Hostname,
		Os:                  report.OS,
		OsVersion:           report.OSVersion,
		Architecture:        report.Architecture,
		AgentVersion:        report.AgentVersion,
		RuntimeCapabilities: report.RuntimeCapabilities,
		ProtocolVersion:     ProtocolVersion,
	}

	if report.CPU != nil {
		req.Cpu = &protov1.CPUInfoProto{
			Model:              report.CPU.Model,
			Vendor:             report.CPU.Vendor,
			Cores:              int32(report.CPU.Cores),
			Threads:            int32(report.CPU.Threads),
			UtilizationPercent: report.CPU.Utilization,
		}
	}

	if report.Memory != nil {
		req.Memory = &protov1.MemoryInfoProto{
			TotalBytes:         report.Memory.TotalBytes,
			AvailableBytes:     report.Memory.AvailableBytes,
			UsedBytes:          report.Memory.UsedBytes,
			UtilizationPercent: report.Memory.UtilizationPercent,
		}
	}

	if report.GPUReport != nil && len(report.GPUReport.GPUs) > 0 {
		for _, g := range report.GPUReport.GPUs {
			req.Gpus = append(req.Gpus, &protov1.GPUInfoProto{
				Vendor:             g.Vendor,
				Name:               g.Name,
				VramTotalBytes:     g.VRAMTotalBytes,
				VramUsedBytes:      g.VRAMUsedBytes,
				VramAvailableBytes: g.VRAMAvailBytes,
				UtilizationPercent: g.UtilizationPercent,
				TemperatureC:       g.TemperatureC,
				PowerUsageWatts:    g.PowerUsageWatts,
				DriverVersion:      g.DriverVersion,
				ComputeCapability:  g.ComputeCapability,
			})
		}
	}

	if report.Storage != nil {
		req.Storage = &protov1.StorageInfoProto{
			TotalBytes:     report.Storage.TotalBytes,
			AvailableBytes: report.Storage.AvailableBytes,
			UsedBytes:      report.Storage.UsedBytes,
			UsedPercent:    report.Storage.UsedPercent,
		}
	}

	if report.Network != nil && len(report.Network.Interfaces) > 0 {
		netProto := &protov1.NetworkInfoProto{
			EstimatedBandwidth: report.Network.EstimatedBandwidth,
		}
		for _, iface := range report.Network.Interfaces {
			netProto.Interfaces = append(netProto.Interfaces, &protov1.NetworkInterfaceProto{
				Name:                  iface.Name,
				Ips:                   iface.IPs,
				Mac:                   iface.MAC,
				IsWired:               iface.IsWired,
				EstimatedBandwidthBps: iface.EstimatedBandwidth,
			})
		}
		req.Network = netProto
	}

	return req
}
