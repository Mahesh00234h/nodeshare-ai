package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	statusFlag := flag.Bool("status", false, "Print Node Agent live status")
	diagFlag := flag.Bool("diagnostics", false, "Print detailed Node Agent diagnostics & hardware report")
	verFlag := flag.Bool("version", false, "Print Node Agent version")
	portFlag := flag.Int("port", 9090, "Local diagnostic API port")
	flag.Parse()

	args := flag.Args()
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}

	if *verFlag || cmd == "version" {
		fmt.Println("Distributed AI Node Agent v1.0.0 (Go 1.22+)")
		return
	}

	if *diagFlag || cmd == "diagnostics" {
		runDiagnostics(*portFlag)
		return
	}

	if *statusFlag || cmd == "status" {
		runStatus(*portFlag)
		return
	}

	// Default help
	fmt.Println("=== Distributed AI Node Agent CLI ===")
	fmt.Println("Usage:")
	fmt.Println("  node-cli --status        # Show live agent status & active jobs")
	fmt.Println("  node-cli --diagnostics   # Show full hardware & node diagnostics")
	fmt.Println("  node-cli --version       # Show agent version")
}

func runStatus(port int) {
	url := fmt.Sprintf("http://127.0.0.1:%d/status", port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error connecting to Node Agent diagnostic API at %s: %v\nIs node-agent service running?\n", url, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	_ = json.Unmarshal(body, &data)

	pretty, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println("=== NODE AGENT STATUS ===")
	fmt.Println(string(pretty))
}

func runDiagnostics(port int) {
	url := fmt.Sprintf("http://127.0.0.1:%d/status", port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error querying diagnostics: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("=== FULL NODE DIAGNOSTICS REPORT ===")
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err == nil {
		pretty, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(pretty))
	} else {
		fmt.Println(string(body))
	}
}
