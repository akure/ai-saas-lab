# Subscription & Entitlement Module — Frontend User Journey & Automated Playwright Testing Guide

**Platform**: AI SaaS Lab — Metering-as-a-Service (MaaS) Infrastructure  
**Package**: `aisaaslab/internal/modules/subscription`  
**File**: `doc/guide_subscription_testing.md`  
**Status**: Production User Guide & Automation Specification  

---

## 1. Overview & UI Architecture

The **Subscription & Entitlement Dashboard** (`SubscriptionTab.tsx`) provides an interactive administrative interface for managing tenant contract lifecycles, governing product plan catalog tiers, inspecting metric entitlements, and simulating real-time Finite State Machine (FSM) state transitions.

```
+-----------------------------------------------------------------------------------+
|                           SUBSCRIPTION DASHBOARD (UI)                             |
|                                                                                   |
|  +--------------------+    +--------------------+    +-------------------------+  |
|  | Contract Inspector |    | FSM Dunning Panel  |    | Plan Catalog Tiers      |  |
|  | (State, Timezone,  |    | (Activate, Cancel, |    | (Starter, Pro,          |  |
|  |  Usability Status) |    |  Payment Fail/Pass)|    |  Custom Tiers)          |  |
|  +---------+----------+    +---------+----------+    +------------+------------+  |
+------------|-------------------------|----------------------------|---------------+
             |                         |                            |
             v                         v                            v
+-----------------------------------------------------------------------------------+
|                          REST API (http://localhost:8080)                         |
|  - GET  /v1/subscription/plans                                                    |
|  - GET  /v1/subscription/{key}                                                    |
|  - POST /v1/subscription/{key}/event                                              |
|  - POST /v1/subscription/contracts                                                |
+-----------------------------------------------------------------------------------+
```

---

## 2. Interactive Features & User Journeys

### 2.1 Contract Inspector & KPI Cards
- **Tenant Key Selector**: Switch between tenant identities (e.g. `sk_lab_pro_8f92a1c4e7b3091d`, `demo-key-starter`).
- **FSM State Indicator**: Color-coded status badges:
  - 🟢 `Active`: Full service access permitted (`is_usable = true`).
  - 🔵 `Trial`: Onboarding trial state (`is_usable = true`).
  - 🟡 `PastDue`: Grace period / restricted usage (`is_usable = false` or warning).
  - 🔴 `Cancelled`: Hard suspension (`is_usable = false`, HTTP 403 Forbidden).
- **Billing Anchor & Timezone**: Displays subscription creation anchor timestamp and validated IANA timezone.

### 2.2 FSM Dunning & Lifecycle Controls
Triggers live FSM transitions via `POST /v1/subscription/{key}/event`:
- **`Activate Tier`**: Fires event `activate` or `payment_succeeded` ➔ transitions state to `Active`.
- **`Payment Failure`**: Fires event `payment_failed` ➔ transitions state to `PastDue`.
- **`Expire Trial`**: Fires event `trial_expired` ➔ transitions state to `PastDue`.
- **`Cancel Access`**: Fires event `cancel` ➔ transitions state to `Cancelled` (suspends service).
- **`Reactivate`**: Fires event `reactivate` ➔ transitions state back to `Active`.

### 2.3 Quota Consumption Simulator
Interactive slider simulating metric unit usage against plan quota bounds (`total_tokens`). Displays a dynamic progress bar that turns amber/rose when usage nears or exceeds plan entitlement limits.

### 2.4 Plan Catalog & Custom Plan Creator Modal
- **Catalog Cards**: Lists available pricing tiers (`Starter`, `Pro`, `Enterprise`, custom plans).
- **Plan Switcher**: One-click contract binding upgrade (`POST /v1/subscription/contracts`).
- **Register Plan Modal**: Form to define custom plan slugs, names, and monthly token quotas.

---

## 3. Automated Playwright Live Browser Testing

The Playwright live browser test script (`client/test_subscription_live.js`) automates the full end-to-end user journey in a real desktop Chromium browser window.

### 3.1 Stack & Prerequisites
- **Node.js**: v18+
- **Playwright**: `@playwright/test` / `playwright`
- **Backend Server**: Go `cmd/lab` binary listening on `http://localhost:8080`
- **Frontend Server**: Vite dev server listening on `http://localhost:3000`

### 3.2 Running the Automation Script

```powershell
# Navigate to client directory
cd "d:\AK-FILES\Golang April\ai-saas-lab\ai-saas-lab\client"

# Run standard live browser test (visible Chromium desktop window)
node test_subscription_live.js

# Run in step-pause mode (pauses for ENTER press between steps)
$env:STEP_PAUSE="true"; node test_subscription_live.js
```

---

## 4. Test Step Specification

| Step | Action | Assertion / Verification | Screenshot Captured |
| :---: | :--- | :--- | :--- |
| **STEP 1** | Navigate to `http://localhost:3000` | Dashboard overview renders with KPI stats | `01_overview_dashboard.png` |
| **STEP 2** | Click `Subscriptions & FSM` sidebar tab | Navigates to Subscription Hub | `02_subscription_fsm_hub.png` |
| **STEP 3** | Type tenant key & click `Inspect` | Contract inspector loads tenant state | `03_inspected_tenant_contract.png` |
| **STEP 4** | Click `Payment Failure` button | State transitions `active` ➔ `past_due` | `04_fsm_past_due_state.png` |
| **STEP 5** | Click `Reactivate` button | State transitions `past_due` / `cancelled` ➔ `active` | `05_fsm_reactivated_state.png` |
| **STEP 6** | Open Register Plan modal & submit | Custom plan registered in catalog | `06_custom_plan_registration_modal.png`<br>`07_custom_plan_registered_catalog.png` |
| **STEP 7** | Click `Switch to Custom Plan` | Tenant contract updated to custom plan | `08_switched_plan_active_contract.png` |

---

## 5. Visual Feedback & Video Recording

1. **Gold Ring Highlight (`#F5CE26`)**: Target element illuminates with a gold outline before every click.
2. **Sky-Blue Ring Highlight (`#38BDF8`)**: Input boxes illuminate with a sky-blue ring before text entry.
3. **Human Typing Speed**: All text inputs type character-by-character with a 60ms delay.
4. **Session Video Recording**: The complete test run is recorded as a `.webm` video file in `client/tests/videos/subscription/`.

---

## 6. Execution Verification Results

```text
====================================================
 🚀 Playwright Live Browser Test: Subscription & FSM
 🔑 Test Suffix: 5165
 🌐 Base URL: http://localhost:3000
====================================================

▶ STEP 1: Navigating to AI SaaS Lab Dashboard...
  📸 Screenshot saved: 01_overview_dashboard.png
  ✅ PASS: Dashboard loaded successfully

▶ STEP 2: Opening Subscriptions & FSM Tab...
  📸 Screenshot saved: 02_subscription_fsm_hub.png
  ✅ PASS: Subscriptions & FSM tab loaded with state cards and FSM controls

▶ STEP 3: Inspecting Tenant Subscription Contract...
  📸 Screenshot saved: 03_inspected_tenant_contract.png
  ✅ PASS: Inspected subscription contract for tenant "sk_lab_pro_8f92a1c4e7b3091d"

▶ STEP 4: Simulating FSM Transition: Payment Failure...
  📸 Screenshot saved: 04_fsm_past_due_state.png
  ✅ PASS: Fired event "payment_failed": state transitioned to Past Due

▶ STEP 5: Simulating FSM Transition: Reactivate...
  📸 Screenshot saved: 05_fsm_reactivated_state.png
  ✅ PASS: Fired event "reactivate": state transitioned back to Active

▶ STEP 6: Registering Custom Subscription Plan...
  📸 Screenshot saved: 06_custom_plan_registration_modal.png
  📸 Screenshot saved: 07_custom_plan_registered_catalog.png
  ✅ PASS: Registered custom subscription plan "Custom Growth Plan (5165)"

▶ STEP 7: Switching Tenant Plan Tier...
  📸 Screenshot saved: 08_switched_plan_active_contract.png
  ✅ PASS: Switched tenant contract to "Custom Growth Plan (5165)"

====================================================
 🎉 ALL SUBSCRIPTION & FSM LIVE BROWSER TESTS PASSED!
====================================================
```
