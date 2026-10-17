import urllib.request
import json
import ssl
import sys

url = "https://llm-cluster.onrender.com/v1/chat/completions"
headers = {
    "Content-Type": "application/json",
    "Authorization": "Bearer sk-cluster-controller-secret-key"
}

# Test 1: Non-streaming completion
payload_non_stream = {
    "model": "mock-llama-3",
    "messages": [
        {"role": "user", "content": "Explain quantum computing in one short sentence."}
    ],
    "stream": False
}

ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

print(f"=== TEST 1: POST non-stream request to {url} ===", flush=True)
req = urllib.request.Request(url, data=json.dumps(payload_non_stream).encode("utf-8"), headers=headers, method="POST")

try:
    with urllib.request.urlopen(req, context=ctx, timeout=60) as resp:
        body = resp.read().decode("utf-8")
        print(f"HTTP Status: {resp.status}", flush=True)
        print("Response Body:", flush=True)
        print(body, flush=True)
        data = json.loads(body)
        print("\n--> LLM ANSWER RECEIVED:", flush=True)
        print(data.get("choices", [{}])[0].get("message", {}).get("content"), flush=True)
except urllib.error.HTTPError as e:
    print(f"HTTP Error {e.code}: {e.reason}", flush=True)
    print(e.read().decode("utf-8"), flush=True)
except Exception as e:
    print(f"Error: {e}", flush=True)

# Test 2: Streaming completion
print(f"\n=== TEST 2: POST stream request to {url} ===", flush=True)
payload_stream = {
    "model": "mock-llama-3",
    "messages": [
        {"role": "user", "content": "Tell me a fun fact about space."}
    ],
    "stream": True
}

req_stream = urllib.request.Request(url, data=json.dumps(payload_stream).encode("utf-8"), headers=headers, method="POST")

try:
    with urllib.request.urlopen(req_stream, context=ctx, timeout=60) as resp:
        print(f"HTTP Status: {resp.status}", flush=True)
        print("Streaming chunks:", flush=True)
        full_text = ""
        for line in resp:
            line_str = line.decode("utf-8").strip()
            if line_str.startswith("data: "):
                chunk_data = line_str[6:]
                if chunk_data == "[DONE]":
                    print("\n[STREAM DONE]", flush=True)
                    break
                try:
                    parsed = json.loads(chunk_data)
                    content = parsed.get("choices", [{}])[0].get("delta", {}).get("content", "")
                    if content:
                        print(content, end="", flush=True)
                        full_text += content
                except Exception:
                    pass
        print(f"\n--> FULL STREAMED LLM ANSWER: {full_text}", flush=True)
except urllib.error.HTTPError as e:
    print(f"HTTP Error {e.code}: {e.reason}", flush=True)
    print(e.read().decode("utf-8"), flush=True)
except Exception as e:
    print(f"Error: {e}", flush=True)

