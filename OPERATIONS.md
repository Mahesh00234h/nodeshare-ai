# Operations & Runbook Guide

---

## 1. Node Management Runbook

### How to Check Online Node Status
Inspect registered compute nodes via administrative API:
```bash
curl -H "Authorization: Bearer dev-secret-key" https://llm-cluster.onrender.com/api/admin/nodes
```

### How to Drain a Node for Maintenance
Prevent new jobs from being assigned to a specific node:
```bash
curl -X POST -H "Authorization: Bearer dev-secret-key" \
  https://llm-cluster.onrender.com/api/admin/nodes/{node_id}/drain
```

---

## 2. Troubleshooting & Recovery

### Symptom: Node Status shows `DEGRADED` or `OFFLINE`
1. Verify `node-agent.exe` process is active on target PC: `Get-Process node-agent`.
2. Ensure outbound network access to `https://llm-cluster.onrender.com:443` is permitted by firewall.
3. Restart agent silently in background:
   ```powershell
   cd d:\decentralized\node-agent
   .\bin\launch-agent.exe
   ```

### Symptom: Job Fails with `RPC_WORKER_DISCONNECTED`
1. Check LAN link between primary PC and worker PC (`192.168.1.50`).
2. Verify `llama-rpc-server.exe` is running on port 50052: `netstat -ano | findstr 50052`.
3. Restart RPC worker: `llama-rpc-server.exe -H 0.0.0.0 -p 50052`.
