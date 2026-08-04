# Auth Module Testing Guide

This guide shows how to test the auth module with HTTP requests from:
- Linux / macOS / WSL
- Windows PowerShell
- A browser

The app listens on port 8080 by default, as defined in [config.env](../config.env).

## 1. Start the app

From the project root, run:

```bash
go run ./cmd/lab
```

You should see the server listening on port 8080.

---

## 2. Create an API key

### Linux / macOS / WSL

```bash
curl -X POST http://localhost:8080/admin/api-keys \
  -H "Content-Type: application/json" \
  -d '{"plan":"pro"}'
```

Expected response:

```json
{"api_key":"demo-pro-abc123","plan":"pro"}
```

### Windows PowerShell

In PowerShell, `curl` is often an alias to `Invoke-WebRequest`, so use `Invoke-RestMethod` instead:

```powershell
$body = '{"plan":"pro"}'
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/admin/api-keys" -ContentType "application/json" -Body $body
```

Expected response:

```json
{"api_key":"demo-pro-abc123","plan":"pro"}
```

### Browser

Open the browser developer console and run:

```javascript
fetch("http://localhost:8080/admin/api-keys", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ plan: "pro" })
})
.then(r => r.json())
.then(console.log);
```

---

## 3. Test the auth flow with a valid key

### Linux / macOS / WSL

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"api_key":"demo-pro-abc123","prompt":"hello from auth test"}'
```

This should return an SSE stream beginning with lines like:

```text
data: Echo:

data: hello
```

### Windows PowerShell

```powershell
$body = '{"api_key":"demo-pro-abc123","prompt":"hello from auth test"}'
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/v1/chat/completions" -ContentType "application/json" -Body $body
```

### Browser

```javascript
fetch("http://localhost:8080/v1/chat/completions", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ api_key: "demo-pro-abc123", prompt: "hello from auth test" })
})
.then(r => r.text())
.then(console.log);
```

---

## 4. Test an invalid key

### Linux / macOS / WSL

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"api_key":"bad-key","prompt":"should fail"}'
```

Expected result:
- HTTP status `403 Forbidden`
- Response body contains an auth error

### Windows PowerShell

```powershell
$body = '{"api_key":"bad-key","prompt":"should fail"}'
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/v1/chat/completions" -ContentType "application/json" -Body $body
```

If you want to see the status code explicitly in PowerShell, use:

```powershell
$resp = Invoke-WebRequest -Method Post -Uri "http://localhost:8080/v1/chat/completions" -ContentType "application/json" -Body $body
$resp.StatusCode
$resp.Content
```

---

## 5. Useful notes

- The auth module validates API keys before the completion route proceeds.
- The admin endpoint is `POST /admin/api-keys`.
- The completion endpoint is `POST /v1/chat/completions`.
- If you are testing locally and want a quick seeded key, the app also seeds demo keys during setup for local development flows.

If you want, I can also add a small section with example `curl` commands for revoking keys or testing the billing/quota layer next.
