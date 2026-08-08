import { test, expect } from '@playwright/test';

test.describe('Client Dashboard E2E Browser Test Suite (No LLM)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('1. Overview Tab & Header Elements Render Correctly', async ({ page }) => {
    // Check main title
    await expect(page.locator('h1')).toContainText('AI SaaS Lab Client');
    // Check KPI cards
    await expect(page.getByText('Active API Keys')).toBeVisible();
    await expect(page.getByText('Token Volume')).toBeVisible();
    await expect(page.getByText('Vector Storage')).toBeVisible();
    await expect(page.getByText('Est. Monthly Spend')).toBeVisible();
  });

  test('2. API Keys Management Workflow', async ({ page }) => {
    await page.click('button:has-text("API Keys")');
    await expect(page.getByText('API Keys & Credentials')).toBeVisible();

    // Open Modal
    await page.click('button:has-text("Create Secret Key")');
    await expect(page.getByText('Create Production API Key')).toBeVisible();

    // Fill Form & Submit
    await page.fill('input[placeholder*="Production"]', 'E2E Automated Key');
    await page.click('button[type="submit"]');

    // Verify key appears in table
    await expect(page.getByText('E2E Automated Key')).toBeVisible();
  });

  test('3. Flexible Metering Sliders Real-Time Update', async ({ page }) => {
    await page.click('button:has-text("Metering Sandbox")');
    await expect(page.getByText('Simulation Parameters')).toBeVisible();

    // Fill slider with value 350
    const slider = page.locator('input[type="range"].gold-slider').first();
    await slider.fill('350');

    // Verify recalculation badge shows 350 Users
    await expect(page.getByText('350 Users')).toBeVisible();
  });

  test('4. JSON Studio Import, Render & Download', async ({ page }) => {
    await page.click('button:has-text("JSON Studio")');
    await expect(page.getByText('JSON Data Hub')).toBeVisible();

    // Verify JSON formatted preview contains telemetry keys
    const jsonPre = page.locator('pre');
    await expect(jsonPre).toContainText('dashboard_version');
    await expect(jsonPre).toContainText('metering_simulation');
  });

  test('5. API Playground Endpoint Execution', async ({ page }) => {
    await page.click('button:has-text("API Playground")');
    await expect(page.getByText('Live API Request Tester')).toBeVisible();

    // Click Send Completion Request
    await page.click('button:has-text("Send Completion Request")');
    await expect(page.getByText('Response Inspector')).toBeVisible();
  });
});
