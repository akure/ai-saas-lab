# AI SaaS Lab - Client Dashboard Architecture & Beginner Guide

Welcome to the **AI SaaS Lab Client Dashboard** documentation. This document is designed for developers, open-source contributors, and beginners who want to understand, run, customize, or extend the production-grade **Commercial Black & Gold** client dashboard.

---

## Table of Contents
1. [High-Level Architecture](#high-level-architecture)
2. [Monorepo Directory Layout](#monorepo-directory-layout)
3. [Beginner Quickstart (Running Backend + Client)](#beginner-quickstart-running-backend--client)
4. [Decoupled REST API Specification](#decoupled-rest-api-specification)
5. [Feature Deep-Dive & Component Guide](#feature-deep-dive--component-guide)
6. [Offline Standalone / Fallback Mode](#offline-standalone--fallback-mode)
7. [How to Customize & Extend](#how-to-customize--extend)
8. [Automated Testing Guide](#automated-testing-guide)

---

## High-Level Architecture

The system uses a **loosely-coupled client-server architecture**:

```mermaid
graph TD
    User([User Browser]) --> ClientApp[React 18 + Vite SPA (Port 3000)]
    
    subgraph Client App Architecture (client/)
        ClientApp --> UIComponents[UI Components / Sliders / Charts]
        UIComponents --> APIService[API Service Layer (services/api.ts)]
        APIService -->|HTTP REST JSON| Proxy[Vite Dev Proxy]
    end

    subgraph Go Backend Kernel (cmd/lab)
        Proxy -->|Port 8080| GoServer[Go REST Server (cmd/lab)]
        GoServer --> AuthModule[auth: API Key Issuance]
        GoServer --> BillingModule[billing: Quotas & Metering]
        GoServer --> CompletionModule[completion: AI Streaming]
    end

    subgraph Offline Mode
        APIService -.->|Fallback if Offline| MockEngine[Mock Telemetry Generator]
    end
```

### Why Decoupled Architecture?
- **Backend Neutrality**: The React frontend communicates strictly over standard JSON REST APIs. The Go backend can be updated, scaled, or replaced without breaking the UI.
- **Standalone Development**: Beginners can run and test the frontend UI even if the Go backend environment is offline.

---

## Monorepo Directory Layout

```
ai-saas-lab/
├── cmd/
│   └── lab/               # Go Backend Entry Point (Port 8080)
├── internal/              # Go Core Modules (auth, billing, completion)
├── client/                # Client Dashboard App (Port 3000)
│   ├── public/            # Static Web Assets
│   ├── src/
│   │   ├── components/    # Modular UI Tabs
│   │   │   ├── Header.tsx                 # Top bar with status & plan badges
│   │   │   ├── Sidebar.tsx                # Commercial Black & Gold tab menu
│   │   │   ├── OverviewTab.tsx            # KPI Summary & System Gauges
│   │   │   ├── ApiKeysTab.tsx             # Secret Key Generator & Snippets
│   │   │   ├── MeteringSimulatorTab.tsx   # Interactive Sliders & Live Cost
│   │   │   ├── AnalyticsStorageTab.tsx    # Traffic & Vector Storage Charts
│   │   │   ├── JsonStudioTab.tsx          # Receive, Render & Download JSON
│   │   │   └── ApiTesterTab.tsx           # Live REST API Request Sandbox
│   │   ├── services/
│   │   │   └── api.ts                     # REST HTTP Client & Offline Engine
│   │   ├── types/
│   │   │   └── index.ts                   # Shared TypeScript Interfaces
│   │   ├── App.tsx                        # Main Layout Shell & Toast Manager
│   │   ├── index.css                      # Commercial Black & Gold Design System
│   │   └── main.tsx                       # React Application Entry Point
│   ├── package.json       # Dependencies & Scripts
│   ├── tailwind.config.js # Commercial Black & Gold Theme Configuration
│   ├── vite.config.ts     # Vite Dev Server & API Proxy Config
│   └── test_dashboard.js  # Standalone Playwright Test Script
└── doc/
    ├── client/            # Client Documentation (This file)
    └── agent-testing/     # Automated Testing & Browser Subagent Docs
```

---

## Beginner Quickstart (Running Backend + Client)

### Prerequisites
1. **Go**: Version 1.21+ ([download](https://go.dev/dl/))
2. **Node.js**: Version 18+ or 20+ ([download](https://nodejs.org/))

### Step 1: Start the Go Backend Server
Open Terminal Terminal 1:
```powershell
# From project root
go run ./cmd/lab
```
*The server starts listening on `http://localhost:8080`.*

### Step 2: Start the Client Dashboard
Open Terminal Terminal 2:
```powershell
# Navigate to client directory
cd client

# Install dependencies (First time only)
npm install

# Start Vite dev server
npm run dev
```
*Open your browser at `http://localhost:3000`.*

---

## Decoupled REST API Specification

The client dashboard interacts with 3 core backend endpoints:

### 1. Issue API Key (`POST /admin/api-keys`)
- **Request Body**:
  ```json
  {
    "plan": "pro"
  }
  ```
- **Response**:
  ```json
  {
    "api_key": "sk_lab_pro_8f92a1c4e7b3091d",
    "status": "created"
  }
  ```

### 2. AI Completion & Metering (`POST /v1/chat/completions`)
- **Request Body**:
  ```json
  {
    "api_key": "sk_lab_pro_8f92a1c4e7b3091d",
    "prompt": "Hello from client dashboard"
  }
  ```
- **Response**:
  ```
  "Processed completion successfully."
  ```

### 3. Fetch Token Usage (`GET /v1/usage/:key`)
- **Response**:
  ```json
  {
    "total_requests": 142,
    "total_tokens": 184500
  }
  ```

---

## Feature Deep-Dive & Component Guide

### 1. API Key Provisioning (`ApiKeysTab.tsx`)
- **Key Generation Modal**: Select plan tier (`Free` = 300 RPM, `Pro` = 2,500 RPM, `Enterprise` = 10,000 RPM).
- **Security Features**: Click-to-copy to clipboard, secret key mask/reveal toggle (`sk_lab_...`), and active/revoked operational status toggle.
- **Code Snippet Generator**: Auto-generates ready-to-run **cURL (Bash)** and **PowerShell** commands tailored to each API key for immediate backend testing.

### 2. Flexible Metering Simulator (`MeteringSimulatorTab.tsx`)
Allows testing mock/simulated traffic loads using dynamic sliders:
- **Sliders Available**:
  - *Concurrent Users* (1 to 1,000)
  - *Request Rate* (1 to 500 req/sec)
  - *Input Prompt Tokens* (100 to 16,000 tokens/req)
  - *Completion Output Tokens* (50 to 8,000 tokens/req)
  - *Vector Storage Index Size* (0.5 to 250 GB)
  - *Cache Hit Ratio* (0% to 95%)
- **Calculated Metrics**: Real-time calculated velocity (tokens/sec), bandwidth (MB/s), estimated daily/monthly cost ($), and vector embeddings count.

### 3. JSON Data Hub (`JsonStudioTab.tsx`)
- **Receive / Import JSON**: Paste raw JSON payload or drag-and-drop a `.json` file to dynamically update dashboard state.
- **Interactive JSON Inspector**: Formatted view, syntax colorization, and key filter search.
- **1-Click JSON Downloads**:
  - `usage_and_metering_stats.json`
  - `api_keys_export.json`
  - `simulation_config.json`

---

## Offline Standalone / Fallback Mode

If the Go backend server is offline or unreachable, `client/src/services/api.ts` automatically catches connection errors and seamlessly switches to **Standalone Simulation Mode**:
- Generates simulated API key secret tokens (`sk_lab_pro_...`).
- Computes mock completion responses and token counts.
- Displays an amber **"Standalone Simulation Mode"** indicator in the header.

---

## How to Customize & Extend

### Changing Theme Colors
Theme tokens are configured in `client/tailwind.config.js`:
```javascript
colors: {
  obsidian: { 950: '#0B0B0E', 900: '#121217' },
  gold: { 500: '#D4AF37', 400: '#F5CE26' }
}
```
Modify these hex values to change primary accents across the dashboard.

### Adding a New Flexible Slider
Open `client/src/components/MeteringSimulatorTab.tsx` and add a new slider input:
```tsx
<input
  type="range"
  min="1"
  max="100"
  value={simParams.myNewParam}
  onChange={(e) => updateField('myNewParam', parseInt(e.target.value))}
  className="gold-slider"
/>
```

---

## Automated Testing Guide

Run Playwright automated E2E tests with **$0 LLM cost**:
```powershell
cd client
npx playwright install chromium
node test_dashboard.js
```
For advanced automated testing documentation, see [doc/agent-testing/browser_agent_testing_guide.md](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/doc/agent-testing/browser_agent_testing_guide.md).
