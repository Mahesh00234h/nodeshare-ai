module github.com/distributed-ai/node-agent

go 1.22

require (
	github.com/distributed-ai/distributed-runtime v0.0.0-00010101000000-000000000000
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/shirou/gopsutil/v3 v3.24.5
	golang.org/x/sys v0.20.0
	google.golang.org/grpc v1.64.0
)

replace github.com/distributed-ai/distributed-runtime => ../distributed-runtime

require (
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)
