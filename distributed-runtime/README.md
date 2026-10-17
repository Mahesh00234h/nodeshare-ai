# Distributed LLM Inference Runtime V1

The **Distributed Inference Runtime V1** enables multiple physical PCs to participate in executing a **single LLM inference request** by distributing model layers across GPUs via `llama.cpp`'s native RPC backend over local Ethernet.

---

## Directory Structure

```
distributed-runtime/
├── cmd/
│   ├── runtime/          # Runtime daemon entrypoint
│   └── runtime-cli/      # Diagnostic & benchmark CLI tool
├── internal/
│   ├── runtime/          # State machine, lifecycle & capabilities
│   ├── llama/            # llama-cli & llama-rpc-server process supervisors
│   ├── execution/        # Heterogeneous planner & VRAM budgeting
│   ├── communication/    # LAN interface detection & security guard
│   ├── model/            # GGUF manifest & checksum manager
│   └── recovery/         # Failure classification & re-planning engine
├── configs/
├── scripts/
├── tests/
└── docs/
```

---

## Building

```powershell
# Build CLI tool
go build -o bin/runtime-cli.exe ./cmd/runtime-cli

# Build Daemon
go build -o bin/runtime.exe ./cmd/runtime
```

---

## Running Tests

```powershell
# Run unit & integration tests
go test -v ./...
```
