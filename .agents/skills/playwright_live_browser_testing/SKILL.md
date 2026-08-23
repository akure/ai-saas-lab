---
name: playwright-live-browser-testing
description: >
  Live, visible browser automation testing for the AI SaaS Lab Tenant Catalog
  Dashboard using Playwright. Tests run in a non-headless Chromium window with
  visual click highlights, human-speed typing, video recording, and step-pause
  breakpoints. Use this skill when you need to run, extend, or debug the
  Tenant Catalog UI/UX test suite.
tags: [testing, playwright, frontend, tenant-catalog, ui, browser-automation]
---

# Playwright Live Browser Testing — AI SaaS Lab

## Overview

This skill documents how to run, extend, and debug the Playwright-based live
browser test suite for the Tenant Catalog Dashboard module of the AI SaaS Lab
MaaS (Metering as a Service) platform.

The test suite:
- Opens a **real visible Chromium window** on the desktop (`headless: false`)
- Uses **`slowMo: 1000ms`** so every action is clearly observable
- **Highlights elements** with gold rings before clicking and blue rings before typing
- **Types character-by-character** at 60ms/char (human-readable speed)
- **Records a `.webm` video** of the full test session for replay
- Takes **numbered screenshots** after each major step
- Supports a **step-pause mode** (manual stepping via `STEP_PAUSE=true` env var)
- Uses **unique timestamped IDs** per run to avoid data collisions

---

## Stack & Prerequisites

| Requirement | Version / Notes |
|---|---|
| Node.js | v18+ (ESM modules, `import` syntax) |
| Playwright | `@playwright/test` or `playwright` package |
| Chromium | Auto-installed by Playwright (`npx playwright install chromium`) |
| Frontend Server | Vite dev server on `http://localhost:3000` |
| Backend Server | Go `cmd/lab` binary on `http://localhost:8080` |

### Install Playwright (from client/ directory)

```powershell
npm.cmd install playwright
npx.cmd playwright install chromium
```

> **Windows Note**: Always use `npm.cmd` and `npx.cmd` in PowerShell to avoid
> script execution policy blocks.

---

## File Locations

```
ai-saas-lab/
└── client/
    ├── test_tenant_catalog_live.js    <- Main live test script (production-grade)
    ├── test_tenant_catalog.js         <- Original headless test (batch screenshots)
    └── tests/
        ├── screenshots/               <- PNG screenshots (numbered, auto-created)
        │   ├── 01_overview_page.png
        │   ├── 02_tenant_catalog_hub.png
        │   └── ... (12 total)
        └── videos/                    <- WebM video recordings (auto-created)
            └── <timestamp>.webm
```

---

## Running the Tests

### Standard live run (browser opens on desktop, auto-advances)
```powershell
cd "d:\AK-FILES\Golang April\ai-saas-lab\ai-saas-lab\client"
node test_tenant_catalog_live.js
```

### Step-pause mode (browser pauses and waits for ENTER between each step)
```powershell
$env:STEP_PAUSE="true"; node test_tenant_catalog_live.js
```

### Headless/CI mode (no window, just screenshots + video)
```
Edit CONFIG.headless = true at the top of the script, then run normally.
```

### Full run checklist before starting

1. **Start the Go backend**:
   ```powershell
   cd "d:\AK-FILES\Golang April\ai-saas-lab\ai-saas-lab"
   go run ./cmd/lab/main.go
   ```
   Verify: `curl.exe http://localhost:8080/healthz`

2. **Start the Vite dev server**:
   ```powershell
   cd client
   npm.cmd run dev
   ```
   Verify: browser opens `http://localhost:3000`

3. **Run the test**:
   ```powershell
   node test_tenant_catalog_live.js
   ```

---

## Test Steps — Full Specification

### STEP 1 — Navigate to Dashboard
- **Action**: `page.goto('http://localhost:3000', { waitUntil: 'networkidle' })`
- **Wait**: 1200ms for full React hydration
- **Screenshot**: `01_overview_page.png`
- **Verify**: Overview page with sidebar, header, and KPI stats visible

### STEP 2 — Open Tenant Catalog Tab
- **Locator**: `button:has-text("Tenant Catalog")` (sidebar button)
- **Action**: Visual highlight (gold ring) + click
- **Wait**: 1500ms for tab transition animation
- **Screenshot**: `02_tenant_catalog_hub.png`
- **Verify**: 4 KPI cards visible (Services, Metrics, Plans, Storage Tier)

### STEP 3 — Register a Service
- **Trigger**: Click `button:has-text("+ Service")`
- **Modal fields**:
  - Service ID: `input[placeholder*="ai-completion"]` -> `ai-voice-{suffix}`
  - Service Name: `input[placeholder*="AI Completion Engine"]` -> `Voice Agent ({suffix})`
  - Description: `textarea[placeholder*="Brief summary"]` -> description text
- **Screenshot**: `03_service_registration_modal.png` (before submit)
- **Submit**: `button[type="submit"]:has-text("Register Service")`
- **Wait**: 1500ms for modal close + list refresh
- **Screenshot**: `04_service_registered_list.png`
- **Backend call**: `POST /v1/tenant/catalog/services` with `X-API-Key: demo-key-starter`

### STEP 4 — Register a Metric
- **Trigger**: Click `button:has-text("+ Metric")`
- **Modal fields**:
  - Metric ID: `input[placeholder*="prompt_tokens"]` -> `audio_tokens_{suffix}`
  - Metric Name: `input[placeholder*="Input Tokens Processed"]` -> `Processed Audio Tokens ({suffix})`
- **Screenshot**: `05_metric_registration_modal.png`
- **Submit**: `button[type="submit"]:has-text("Register Metric")`
- **Wait**: 1500ms
- **Backend call**: `POST /v1/tenant/catalog/metrics`

### STEP 5 — Register a Plan
- **Trigger**: Click `button:has-text("+ Plan")`
- **Modal fields**:
  - Plan ID: `input[placeholder*="pro_tier"]` -> `voice_pro_{suffix}`
  - Plan Name: `input[placeholder*="Pro Growth"]` -> `Voice AI Scale Plan ({suffix})`
- **Screenshot**: `06_plan_registration_modal.png`
- **Submit**: `button[type="submit"]:has-text("Register Plan")`
- **Wait**: 1800ms
- **Backend call**: `POST /v1/tenant/catalog/plans`

### STEP 6 — Dependency Graph
- **Trigger**: Click `button:has-text("Dependency Graph")`
- **Wait**: 2000ms for graph rendering
- **Screenshot**: `07_dependency_graph.png`
- **Verify**: Service -> Metric -> Plan tree visible

### STEP 7 — JSON Studio & SDK
- **Trigger**: Click `button:has-text("JSON Studio & SDK")`
- **Wait**: 1200ms
- **Screenshot**: `08_json_studio_schema.png`
- Click `button:has-text("TypeScript SDK")` sub-tab
- **Screenshot**: `09_typescript_sdk_view.png`
- Click `button:has-text("cURL API")` sub-tab
- **Screenshot**: `10_curl_api_view.png`

### STEP 8 — Metering Calculator Playground
- **Trigger**: Click `button:has-text("Catalog Explorer")` to return to explorer
- **Trigger**: Click `button:has-text("Simulate")` on any plan card
- **Modal field**: `input[type="number"]` -> `150000` (tokens)
- **Trigger**: Click `button:has-text("Simulate API Metering Event Emission")`
- **Wait**: 2000ms (backend telemetry event fires)
- **Screenshot**: `11_metering_calculator_modal.png`
- **Close**: `page.keyboard.press('Escape')`

### STEP 9 — Search & Filter
- **Locator**: `input[placeholder*="Search by ID"]`
- **Type**: the service ID from Step 3 (`ai-voice-{suffix}`)
- **Wait**: 1500ms for reactive filter
- **Screenshot**: `12_search_filter_result.png`
- **Verify**: Only the matching service card is shown

---

## Unique ID Strategy

Every test run generates a 4-digit suffix from the last 4 digits of `Date.now()`:

```js
const testSuffix = Date.now().toString().slice(-4);
const ids = {
  service: `ai-voice-${testSuffix}`,
  metric:  `audio_tokens_${testSuffix}`,
  plan:    `voice_pro_${testSuffix}`,
};
```

This ensures each run creates fresh resources and never hits 409 Conflict errors.

---

## Visual Feedback System

### Click highlighting (Gold ring)
Before every click, the target element gets a gold outline + glow:
```js
el.style.outline = '3px solid #F5CE26';
el.style.boxShadow = '0 0 20px #D4AF37';
```

### Type highlighting (Blue ring)
Before typing into any input, it gets a sky-blue ring:
```js
el.style.outline = '3px solid #38BDF8';
el.style.boxShadow = '0 0 15px #38BDF8';
```

### Human-speed typing
All `locator.type()` calls use `{ delay: 60 }` — 60ms per character.

---

## Video Recording

Playwright records the entire session as `.webm` via `recordVideo`:

```js
const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  recordVideo: {
    dir: CONFIG.videoDir,   // client/tests/videos/
    size: { width: 1440, height: 900 },
  },
});
```

> **Critical**: Video is only saved when `context.close()` is called — BEFORE
> `browser.close()`. Always do this in the `finally` block.

---

## Window Positioning

To guarantee the Chromium window appears at the front of the desktop:

```js
args: [
  '--window-size=1440,900',
  '--window-position=0,0',   // Top-left corner of primary monitor
],
```

---

## Backend API Key Setup

`cmd/lab/main.go` seeds demo keys **unconditionally on every startup**:

```go
// Always seed demo keys in memory for lab environment
_ = authMod.Service().RegisterAPIKey("demo-key-starter", "starter")
_ = authMod.Service().RegisterAPIKey("demo-key-pro", "pro")
_ = authMod.Service().RegisterAPIKey("sk_lab_pro_8f92a1c4e7b3091d", "pro")
_ = authMod.Service().RegisterAPIKey("sk_lab_free_3c7d91e8b2a5", "free")
```

The frontend uses `demo-key-starter` as the default key in `catalogApi.ts`.

---

## Catalog API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET`  | `/v1/tenant/catalog/overview` | KPI stats |
| `GET`  | `/v1/tenant/catalog/services` | List services |
| `POST` | `/v1/tenant/catalog/services` | Register service |
| `GET`  | `/v1/tenant/catalog/metrics`  | List metrics |
| `POST` | `/v1/tenant/catalog/metrics`  | Register metric |
| `GET`  | `/v1/tenant/catalog/plans`    | List plans |
| `POST` | `/v1/tenant/catalog/plans`    | Register plan |

All require `X-API-Key` header.

---

## Debugging Runbook

### Browser window not visible
- **Cause**: Opened behind other apps or on secondary monitor
- **Fix**: `--window-position=0,0` is set; check Windows taskbar for Chromium icon

### `locator.click: Timeout` — modal backdrop intercepts
- **Cause**: A modal's `fixed inset-0` overlay is blocking the click target behind it
- **Fix**: Always close any open modal before clicking outside it:
  ```js
  await page.keyboard.press('Escape');
  await page.waitForTimeout(500);
  ```

### 409 Conflict from backend on re-run
- **Cause**: Same ID already registered in memory from last run
- **Fix**: The `Date.now().slice(-4)` suffix makes every run unique automatically

### `{"error":"invalid or revoked api key"}`
- **Cause**: Backend restarted and lost in-memory keys (pre-fix behavior)
- **Fix**: Confirm `cmd/lab/main.go` has the unconditional seed block OUTSIDE the migration `Up` function

### `npm` / script execution policy error in PowerShell
- **Fix**: Use `npm.cmd` and `npx.cmd` on Windows PowerShell

### Video file not created
- **Cause**: `browser.close()` called before `context.close()`
- **Fix**: Order must be: `await context.close()` THEN `await browser.close()`

---

## Extending the Test Suite

### Add a new test step
1. Add async logic in sequence inside the `try` block
2. Call `await shot(page, nextNum, 'step_name')` for screenshot
3. Call `pass('Step Name')` on success
4. Add `await maybePause('Step Name')` for pause-mode support

### Add a new locator
Prefer resilient matchers over brittle class names:
```js
// Good
page.locator('button:has-text("Register Service")')
page.locator('input[placeholder*="ai-completion"]')

// Avoid
page.locator('.btn-primary.submit-btn')
```

### Run only specific steps
```js
// Wrap steps in env-var guard
if (process.env.STEPS?.includes('3')) {
  // Step 3 logic
}
```
```powershell
$env:STEPS="3,4,5"; node test_tenant_catalog_live.js
```
