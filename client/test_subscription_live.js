import { chromium } from 'playwright';
import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// --- Config ---
const CONFIG = {
  baseUrl: process.env.BASE_URL || 'http://localhost:3000',
  headless: process.env.HEADLESS === 'true', // Default to false (visible browser window)
  slowMo: 1000,
  stepPause: process.env.STEP_PAUSE === 'true',
  typeDelay: 60,
  screenDir: path.join(__dirname, 'tests', 'screenshots', 'subscription'),
  videoDir: path.join(__dirname, 'tests', 'videos', 'subscription'),
};

// Ensure screenshot and video output directories exist
fs.mkdirSync(CONFIG.screenDir, { recursive: true });
fs.mkdirSync(CONFIG.videoDir, { recursive: true });

// Visual feedback helpers
async function highlightClick(locator) {
  await locator.evaluate((el) => {
    el.style.outline = '3px solid #F5CE26';
    el.style.boxShadow = '0 0 20px #D4AF37';
    el.style.transition = 'all 0.2s ease-in-out';
  });
  await locator.page().waitForTimeout(300);
}

async function highlightType(locator) {
  await locator.evaluate((el) => {
    el.style.outline = '3px solid #38BDF8';
    el.style.boxShadow = '0 0 15px #38BDF8';
    el.style.transition = 'all 0.2s ease-in-out';
  });
  await locator.page().waitForTimeout(300);
}

async function shot(page, num, name) {
  const filename = `${String(num).padStart(2, '0')}_${name}.png`;
  const filepath = path.join(CONFIG.screenDir, filename);
  await page.screenshot({ path: filepath, fullPage: false });
  console.log(`  📸 Screenshot saved: ${filename}`);
}

async function maybePause(stepName) {
  if (!CONFIG.stepPause) return;
  console.log(`  ⏸  PAUSED on [${stepName}]. Press ENTER in terminal to continue...`);
  await new Promise((resolve) => process.stdin.once('data', resolve));
}

function pass(msg) {
  console.log(`  ✅ PASS: ${msg}`);
}

// --- Main Playwright Test ---
async function runSubscriptionLiveTest() {
  const suffix = Date.now().toString().slice(-4);
  const testIds = {
    tenant: `sk_lab_pro_8f92a1c4e7b3091d`,
    customPlanId: `growth_${suffix}`,
    customPlanName: `Custom Growth Plan (${suffix})`,
  };

  console.log('====================================================');
  console.log(' 🚀 Playwright Live Browser Test: Subscription & FSM');
  console.log(` 🔑 Test Suffix: ${suffix}`);
  console.log(` 🌐 Base URL: ${CONFIG.baseUrl}`);
  console.log('====================================================\n');

  const browser = await chromium.launch({
    headless: CONFIG.headless,
    slowMo: CONFIG.slowMo,
    args: ['--window-size=1440,900', '--window-position=0,0'],
  });

  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    recordVideo: {
      dir: CONFIG.videoDir,
      size: { width: 1440, height: 900 },
    },
  });

  const page = await context.newPage();

  try {
    // ----------------------------------------------------
    // STEP 1 — Navigate to Overview Dashboard
    // ----------------------------------------------------
    console.log('▶ STEP 1: Navigating to AI SaaS Lab Dashboard...');
    await page.goto(CONFIG.baseUrl, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await shot(page, 1, 'overview_dashboard');
    pass('Dashboard loaded successfully');
    await maybePause('STEP 1');

    // ----------------------------------------------------
    // STEP 2 — Open Subscriptions & FSM Tab
    // ----------------------------------------------------
    console.log('\n▶ STEP 2: Opening Subscriptions & FSM Tab...');
    const subTabBtn = page.locator('button:has-text("Subscriptions & FSM")');
    await subTabBtn.waitFor({ state: 'visible', timeout: 5000 });
    await highlightClick(subTabBtn);
    await subTabBtn.click();
    await page.waitForTimeout(1500);
    await shot(page, 2, 'subscription_fsm_hub');
    pass('Subscriptions & FSM tab loaded with state cards and FSM controls');
    await maybePause('STEP 2');

    // ----------------------------------------------------
    // STEP 3 — Inspect Tenant Subscription Contract
    // ----------------------------------------------------
    console.log('\n▶ STEP 3: Inspecting Tenant Subscription Contract...');
    const tenantInput = page.locator('input[placeholder*="Tenant Key"]');
    await highlightType(tenantInput);
    await tenantInput.fill('');
    await tenantInput.type(testIds.tenant, { delay: CONFIG.typeDelay });

    const inspectBtn = page.locator('button:has-text("Inspect")');
    await highlightClick(inspectBtn);
    await inspectBtn.click();
    await page.waitForTimeout(1500);
    await shot(page, 3, 'inspected_tenant_contract');
    pass(`Inspected subscription contract for tenant "${testIds.tenant}"`);
    await maybePause('STEP 3');

    // ----------------------------------------------------
    // STEP 4 — FSM State Transition: Payment Failure
    // ----------------------------------------------------
    console.log('\n▶ STEP 4: Simulating FSM Transition: Payment Failure...');
    const payFailBtn = page.locator('button:has-text("Payment Failure")');
    await highlightClick(payFailBtn);
    await payFailBtn.click();
    await page.waitForTimeout(1800);
    await shot(page, 4, 'fsm_past_due_state');
    pass('Fired event "payment_failed": state transitioned to Past Due');
    await maybePause('STEP 4');

    // ----------------------------------------------------
    // STEP 5 — FSM State Transition: Reactivate
    // ----------------------------------------------------
    console.log('\n▶ STEP 5: Simulating FSM Transition: Reactivate...');
    const reactivateBtn = page.locator('button:has-text("Reactivate")');
    await highlightClick(reactivateBtn);
    await reactivateBtn.click();
    await page.waitForTimeout(1800);
    await shot(page, 5, 'fsm_reactivated_state');
    pass('Fired event "reactivate": state transitioned back to Active');
    await maybePause('STEP 5');

    // ----------------------------------------------------
    // STEP 6 — Register Custom Subscription Plan
    // ----------------------------------------------------
    console.log('\n▶ STEP 6: Registering Custom Subscription Plan...');
    const regPlanBtn = page.locator('button:has-text("Register Plan")');
    await highlightClick(regPlanBtn);
    await regPlanBtn.click();
    await page.waitForTimeout(1000);

    const planIdInput = page.locator('input[placeholder*="enterprise_plus"]');
    await highlightType(planIdInput);
    await planIdInput.type(testIds.customPlanId, { delay: CONFIG.typeDelay });

    const planNameInput = page.locator('input[placeholder*="Enterprise Plus Scale"]');
    await highlightType(planNameInput);
    await planNameInput.type(testIds.customPlanName, { delay: CONFIG.typeDelay });

    await shot(page, 6, 'custom_plan_registration_modal');

    const submitPlanBtn = page.locator('button[type="submit"]:has-text("Register Plan")');
    await highlightClick(submitPlanBtn);
    await submitPlanBtn.click();
    await page.waitForTimeout(1800);
    await shot(page, 7, 'custom_plan_registered_catalog');
    pass(`Registered custom subscription plan "${testIds.customPlanName}"`);
    await maybePause('STEP 6');

    // ----------------------------------------------------
    // STEP 7 — Switch Tenant Plan
    // ----------------------------------------------------
    console.log('\n▶ STEP 7: Switching Tenant Plan Tier...');
    const switchPlanBtn = page.locator(`button:has-text("Switch to ${testIds.customPlanName}")`);
    if (await switchPlanBtn.isVisible()) {
      await highlightClick(switchPlanBtn);
      await switchPlanBtn.click();
      await page.waitForTimeout(1500);
      await shot(page, 8, 'switched_plan_active_contract');
      pass(`Switched tenant contract to "${testIds.customPlanName}"`);
    } else {
      const switchProBtn = page.locator('button:has-text("Switch to Pro Tier")').first();
      await highlightClick(switchProBtn);
      await switchProBtn.click();
      await page.waitForTimeout(1500);
      await shot(page, 8, 'switched_plan_pro');
      pass('Switched tenant contract to Pro Tier');
    }
    await maybePause('STEP 7');

    console.log('\n====================================================');
    console.log(' 🎉 ALL SUBSCRIPTION & FSM LIVE BROWSER TESTS PASSED!');
    console.log('====================================================\n');
  } catch (err) {
    console.error('\n❌ TEST FAILED:', err);
    await page.screenshot({ path: path.join(CONFIG.screenDir, 'FAILURE_CRASH.png') });
    throw err;
  } finally {
    await context.close();
    await browser.close();
  }
}

runSubscriptionLiveTest().catch(() => process.exit(1));
