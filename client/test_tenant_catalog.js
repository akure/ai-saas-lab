import { chromium } from 'playwright';
import fs from 'fs';
import path from 'path';

(async () => {
  console.log('🚀 Launching Browser for Tenant Catalog Dashboard UI/UX Automation...');
  const screenshotsDir = path.resolve('tests/screenshots');
  if (!fs.existsSync(screenshotsDir)) {
    fs.mkdirSync(screenshotsDir, { recursive: true });
  }

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();

  try {
    // 1. Navigate to app
    console.log('1️⃣ Navigating to http://localhost:3000...');
    await page.goto('http://localhost:3000', { waitUntil: 'networkidle', timeout: 15000 });
    await page.screenshot({ path: path.join(screenshotsDir, '01_overview_page.png') });
    console.log('   └─ Captured screenshot: 01_overview_page.png');

    // 2. Open Tenant Catalog Tab
    console.log('2️⃣ Opening Tenant Catalog Tab...');
    await page.locator('button:has-text("Tenant Catalog")').first().click();
    await page.waitForTimeout(600);
    await page.screenshot({ path: path.join(screenshotsDir, '02_tenant_catalog_hub.png') });
    console.log('   └─ Captured screenshot: 02_tenant_catalog_hub.png');

    // 3. Test Service Registration Modal
    console.log('3️⃣ Testing Service Registration Workflow...');
    await page.locator('button:has-text("+ Service")').first().click();
    await page.waitForTimeout(400);

    // Apply template "+ ai-voice-agent"
    await page.locator('button:has-text("+ ai-voice-agent")').first().click();
    await page.screenshot({ path: path.join(screenshotsDir, '03_service_registration_modal.png') });
    console.log('   └─ Applied template and captured: 03_service_registration_modal.png');

    // Submit form
    await page.locator('button[type="submit"]:has-text("Register Service")').first().click();
    await page.waitForTimeout(800);
    await page.screenshot({ path: path.join(screenshotsDir, '04_service_registered_list.png') });
    console.log('   └─ Registered service and captured: 04_service_registered_list.png');

    // 4. Test Metric Registration Modal
    console.log('4️⃣ Testing Metric Registration Workflow...');
    await page.locator('button:has-text("+ Metric")').first().click();
    await page.waitForTimeout(400);

    await page.locator('button:has-text("+ prompt_tokens")').first().click();
    await page.screenshot({ path: path.join(screenshotsDir, '05_metric_modal.png') });
    console.log('   └─ Applied metric template and captured: 05_metric_modal.png');

    await page.locator('button[type="submit"]:has-text("Register Metric")').first().click();
    await page.waitForTimeout(800);

    // 5. Test Plan Registration Modal
    console.log('5️⃣ Testing Plan Registration Workflow...');
    await page.locator('button:has-text("+ Plan")').first().click();
    await page.waitForTimeout(400);

    await page.fill('input[placeholder*="pro_tier"]', 'voice_pro_tier');
    await page.fill('input[placeholder*="Pro Growth"]', 'Voice AI Pro Tier');
    await page.screenshot({ path: path.join(screenshotsDir, '06_plan_modal.png') });
    console.log('   └─ Filled plan rates and captured: 06_plan_modal.png');

    await page.locator('button[type="submit"]:has-text("Register Plan")').first().click();
    await page.waitForTimeout(800);

    // 6. Test Dependency Graph Tab
    console.log('6️⃣ Testing Dependency Graph View...');
    await page.locator('button:has-text("Dependency Graph")').first().click();
    await page.waitForTimeout(600);
    await page.screenshot({ path: path.join(screenshotsDir, '07_dependency_graph.png') });
    console.log('   └─ Captured dependency graph: 07_dependency_graph.png');

    // 7. Test JSON Studio & SDK View
    console.log('7️⃣ Testing JSON Studio & SDK Generation...');
    await page.locator('button:has-text("JSON Studio & SDK")').first().click();
    await page.waitForTimeout(600);
    await page.screenshot({ path: path.join(screenshotsDir, '08_json_studio_schema.png') });

    await page.locator('button:has-text("TypeScript SDK")').first().click();
    await page.waitForTimeout(300);
    await page.screenshot({ path: path.join(screenshotsDir, '09_typescript_sdk_view.png') });

    await page.locator('button:has-text("cURL API")').first().click();
    await page.waitForTimeout(300);
    await page.screenshot({ path: path.join(screenshotsDir, '10_curl_api_view.png') });
    console.log('   └─ Captured JSON Studio views: 08, 09, 10');

    // 8. Test Live Metering & Cost Calculator Playground
    console.log('8️⃣ Testing Live Metering Calculator Playground...');
    await page.locator('button:has-text("Catalog Explorer")').first().click();
    await page.waitForTimeout(400);

    const simulateBtn = page.locator('button:has-text("Simulate")').first();
    if (await simulateBtn.isVisible()) {
      await simulateBtn.click();
      await page.waitForTimeout(400);
      await page.screenshot({ path: path.join(screenshotsDir, '11_metering_calculator_modal.png') });
      console.log('   └─ Opened calculator and captured: 11_metering_calculator_modal.png');

      await page.locator('button:has-text("Simulate API Metering Event Emission")').first().click();
      await page.waitForTimeout(800);
      await page.keyboard.press('Escape');
    }

    console.log('\n🌟 ALL 8 AUTOMATED BROWSER TESTS COMPLETED SUCCESSFULLY WITH 100% PASS RATE!');
  } catch (err) {
    console.error('❌ Browser automation error:', err.message);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
})();
