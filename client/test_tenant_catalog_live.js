import { chromium } from 'playwright';
import { mkdirSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// ─── CONFIG ────────────────────────────────────────────────────────────────
const CONFIG = {
  headless: false,           // Always show browser window on desktop
  slowMo: 1000,              // 1s delay between every action for visibility
  baseUrl: 'http://localhost:3000',
  screenshotDir: join(__dirname, 'tests', 'screenshots'),
  videoDir: join(__dirname, 'tests', 'videos'),
  windowPosition: '0,0',    // Force window to top-left of screen
  windowSize: '1440,900',
  // Set STEP_PAUSE=true as env var to pause and wait for manual keypress between steps
  stepPause: process.env.STEP_PAUSE === 'true',
};

// ─── HELPERS ────────────────────────────────────────────────────────────────

/**
 * Highlight an element with a gold ring and scroll it into view, then click it.
 * This makes every click visually obvious on screen.
 */
const visibleClick = async (page, locator, label) => {
  console.log(`\n👉 [CLICK]: "${label}"`);
  await locator.scrollIntoViewIfNeeded();
  await locator.evaluate((el) => {
    el.style.outline = '3px solid #F5CE26';
    el.style.boxShadow = '0 0 20px #D4AF37';
    el.style.transition = 'all 0.3s ease';
  });
  await page.waitForTimeout(500);
  await locator.click();
};

/**
 * Highlight an input with a blue ring and type text character-by-character
 * at a realistic human speed (60ms per char) so the user can read what's being typed.
 */
const visibleType = async (page, locator, text, label) => {
  console.log(`⌨️  [TYPE]: "${text}" → ${label}`);
  await locator.scrollIntoViewIfNeeded();
  await locator.evaluate((el) => {
    el.style.outline = '3px solid #38BDF8';
    el.style.boxShadow = '0 0 15px #38BDF8';
  });
  await page.waitForTimeout(300);
  await locator.fill('');
  await locator.type(text, { delay: 60 });
};

/**
 * Take a numbered screenshot and log the filename.
 */
const shot = async (page, num, name) => {
  const filename = `${String(num).padStart(2, '0')}_${name}.png`;
  await page.screenshot({ path: join(CONFIG.screenshotDir, filename), fullPage: false });
  console.log(`   📸 Screenshot saved: ${filename}`);
};

/**
 * If STEP_PAUSE mode is enabled, wait for user to press Enter before continuing.
 * This lets a human step through the test manually at their own pace.
 */
const maybePause = async (stepName) => {
  if (CONFIG.stepPause) {
    process.stdout.write(`\n⏸️  [PAUSED] After: "${stepName}" — press ENTER to continue...`);
    await new Promise((resolve) => process.stdin.once('data', resolve));
  }
};

// ─── MAIN TEST ───────────────────────────────────────────────────────────────
(async () => {
  // Unique suffix to avoid ID collisions across test runs
  const testSuffix = Date.now().toString().slice(-4);
  const ids = {
    service:  `ai-voice-${testSuffix}`,
    metric:   `audio_tokens_${testSuffix}`,
    plan:     `voice_pro_${testSuffix}`,
  };

  console.log('═══════════════════════════════════════════════════════════════');
  console.log('  🚀 AI SaaS Lab — Tenant Catalog Live Browser Test Suite');
  console.log(`  Mode: headless=${CONFIG.headless}, slowMo=${CONFIG.slowMo}ms`);
  console.log(`  Test IDs: service=${ids.service}, metric=${ids.metric}, plan=${ids.plan}`);
  console.log('═══════════════════════════════════════════════════════════════\n');

  mkdirSync(CONFIG.screenshotDir, { recursive: true });
  mkdirSync(CONFIG.videoDir, { recursive: true });

  const browser = await chromium.launch({
    headless: CONFIG.headless,
    slowMo: CONFIG.slowMo,
    args: [
      `--window-size=${CONFIG.windowSize}`,
      `--window-position=${CONFIG.windowPosition}`,  // Snap to top-left corner so user always sees it
      '--no-default-browser-check',
      '--disable-extensions',
    ],
  });

  // Record a .webm video of the entire test session
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    recordVideo: {
      dir: CONFIG.videoDir,
      size: { width: 1440, height: 900 },
    },
  });

  const page = await context.newPage();
  const results = [];

  const pass = (step) => { results.push({ step, status: '✅ PASS' }); };
  const fail = (step, err) => { results.push({ step, status: `❌ FAIL: ${err}` }); };

  try {
    // ─── STEP 1: Navigate to Dashboard ─────────────────────────────────────
    console.log('\n━━━ STEP 1: Navigate to Dashboard ━━━');
    await page.goto(CONFIG.baseUrl, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1200);
    await shot(page, 1, 'overview_page');
    pass('Navigate to Dashboard');
    await maybePause('Navigate to Dashboard');

    // ─── STEP 2: Open Tenant Catalog Tab ───────────────────────────────────
    console.log('\n━━━ STEP 2: Open Tenant Catalog Tab ━━━');
    await visibleClick(page, page.locator('button:has-text("Tenant Catalog")').first(), 'Tenant Catalog Sidebar Tab');
    await page.waitForTimeout(1500);
    await shot(page, 2, 'tenant_catalog_hub');
    pass('Open Tenant Catalog Tab');
    await maybePause('Open Tenant Catalog Tab');

    // ─── STEP 3: Register a Service ────────────────────────────────────────
    console.log('\n━━━ STEP 3: Register a Service ━━━');
    await visibleClick(page, page.locator('button:has-text("+ Service")').first(), '+ Service Button');
    await page.waitForTimeout(800);

    await visibleType(page, page.locator('input[placeholder*="ai-completion"]').first(), ids.service, 'Service ID field');
    await visibleType(page, page.locator('input[placeholder*="AI Completion Engine"]').first(), `Voice Agent (${testSuffix})`, 'Service Name field');
    await visibleType(page, page.locator('textarea[placeholder*="Brief summary"]').first(), 'Ultra-low latency streaming voice inference & token metering engine', 'Service Description textarea');
    await shot(page, 3, 'service_registration_modal');

    await visibleClick(page, page.locator('button[type="submit"]:has-text("Register Service")').first(), 'Register Service Submit Button');
    await page.waitForTimeout(1500);
    await shot(page, 4, 'service_registered_list');
    pass('Register a Service');
    await maybePause('Register a Service');

    // ─── STEP 4: Register a Metric ─────────────────────────────────────────
    console.log('\n━━━ STEP 4: Register a Metric ━━━');
    await visibleClick(page, page.locator('button:has-text("+ Metric")').first(), '+ Metric Button');
    await page.waitForTimeout(800);

    await visibleType(page, page.locator('input[placeholder*="prompt_tokens"]').first(), ids.metric, 'Metric ID field');
    await visibleType(page, page.locator('input[placeholder*="Input Tokens Processed"]').first(), `Processed Audio Tokens (${testSuffix})`, 'Metric Name field');
    await shot(page, 5, 'metric_registration_modal');

    await visibleClick(page, page.locator('button[type="submit"]:has-text("Register Metric")').first(), 'Register Metric Submit Button');
    await page.waitForTimeout(1500);
    pass('Register a Metric');
    await maybePause('Register a Metric');

    // ─── STEP 5: Register a Plan ───────────────────────────────────────────
    console.log('\n━━━ STEP 5: Register a Plan ━━━');
    await visibleClick(page, page.locator('button:has-text("+ Plan")').first(), '+ Plan Button');
    await page.waitForTimeout(800);

    await visibleType(page, page.locator('input[placeholder*="pro_tier"]').first(), ids.plan, 'Plan ID field');
    await visibleType(page, page.locator('input[placeholder*="Pro Growth"]').first(), `Voice AI Scale Plan (${testSuffix})`, 'Plan Display Name field');
    await shot(page, 6, 'plan_registration_modal');

    await visibleClick(page, page.locator('button[type="submit"]:has-text("Register Plan")').first(), 'Register Plan Submit Button');
    await page.waitForTimeout(1800);
    pass('Register a Plan');
    await maybePause('Register a Plan');

    // ─── STEP 6: Dependency Graph ──────────────────────────────────────────
    console.log('\n━━━ STEP 6: Dependency Graph ━━━');
    await visibleClick(page, page.locator('button:has-text("Dependency Graph")').first(), 'Dependency Graph Tab');
    await page.waitForTimeout(2000);
    await shot(page, 7, 'dependency_graph');
    pass('View Dependency Graph');
    await maybePause('View Dependency Graph');

    // ─── STEP 7: JSON Studio & SDK ─────────────────────────────────────────
    console.log('\n━━━ STEP 7: JSON Studio & SDK ━━━');
    await visibleClick(page, page.locator('button:has-text("JSON Studio & SDK")').first(), 'JSON Studio Tab');
    await page.waitForTimeout(1200);
    await shot(page, 8, 'json_studio_schema');

    await visibleClick(page, page.locator('button:has-text("TypeScript SDK")').first(), 'TypeScript SDK Sub-tab');
    await page.waitForTimeout(1200);
    await shot(page, 9, 'typescript_sdk_view');

    await visibleClick(page, page.locator('button:has-text("cURL API")').first(), 'cURL API Sub-tab');
    await page.waitForTimeout(1200);
    await shot(page, 10, 'curl_api_view');
    pass('JSON Studio & SDK Explorer');
    await maybePause('JSON Studio & SDK Explorer');

    // ─── STEP 8: Metering Calculator Playground ────────────────────────────
    console.log('\n━━━ STEP 8: Metering Calculator Playground ━━━');
    await visibleClick(page, page.locator('button:has-text("Catalog Explorer")').first(), 'Catalog Explorer Tab');
    await page.waitForTimeout(1000);

    const simulateBtn = page.locator('button:has-text("Simulate")').first();
    if (await simulateBtn.isVisible()) {
      await visibleClick(page, simulateBtn, 'Simulate Plan Pricing Calculator');
      await page.waitForTimeout(1200);

      const tokenInput = page.locator('input[type="number"]').first();
      if (await tokenInput.isVisible()) {
        await visibleType(page, tokenInput, '150000', 'Simulated Monthly Token Usage');
        await page.waitForTimeout(1000);
      }

      await visibleClick(page, page.locator('button:has-text("Simulate API Metering Event Emission")').first(), 'Simulate Telemetry Emission Button');
      await page.waitForTimeout(2000);
      await shot(page, 11, 'metering_calculator_modal');
      await page.keyboard.press('Escape');
      await page.waitForTimeout(800);
    }
    pass('Metering Calculator Playground');
    await maybePause('Metering Calculator Playground');

    // ─── STEP 9: Search & Filter ───────────────────────────────────────────
    console.log('\n━━━ STEP 9: Search & Date Filter ━━━');
    const searchInput = page.locator('input[placeholder*="Search by ID"]');
    await visibleType(page, searchInput, ids.service, 'Search by ID filter');
    await page.waitForTimeout(1500);
    await shot(page, 12, 'search_filter_result');
    pass('Search & Date Filter');
    await maybePause('Search & Date Filter');

    // ─── FINAL: Leave dashboard visible for 4 seconds ──────────────────────
    console.log('\n\n═══════════════════════════════════════════════════════════════');
    console.log('  📊 TEST RESULTS SUMMARY');
    console.log('═══════════════════════════════════════════════════════════════');
    results.forEach((r) => console.log(`  ${r.status}  →  ${r.step}`));
    const passed = results.filter((r) => r.status.startsWith('✅')).length;
    console.log(`\n  ${passed}/${results.length} steps passed`);
    console.log('═══════════════════════════════════════════════════════════════\n');

    console.log('🎉 Test complete — leaving browser open for 4 seconds...');
    await page.waitForTimeout(4000);

  } catch (err) {
    console.error('\n❌ Unexpected test error:', err.message);
    await shot(page, 99, 'error_state').catch(() => {});
  } finally {
    await context.close(); // This finalizes the video recording
    await browser.close();
    console.log(`\n🎬 Video saved to: ${CONFIG.videoDir}`);
    console.log(`📁 Screenshots in: ${CONFIG.screenshotDir}`);
  }
})();
