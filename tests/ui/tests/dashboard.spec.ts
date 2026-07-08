import { test, expect } from '@playwright/test';

// The dashboard polls /api/v1/{stats,jobs,targets} every 5s. The vmprober under
// test probes two TCP targets (one up = its own :8429, one down = :9) every 2s,
// so the up/down states settle within a few seconds. Assertions below are
// web-first (auto-retrying) — no fixed sleeps.

test.beforeEach(async ({ page }) => {
  await page.goto('/');
});

test('dashboard shell renders', async ({ page }) => {
  await expect(page).toHaveTitle(/VMProber/i);
  await expect(page.locator('.vm-header__title')).toBeVisible();
  // Nav links to the three views.
  await expect(page.locator('.vm-header__nav a[data-page="dashboard"]')).toBeVisible();
  await expect(page.locator('#refreshBtn')).toBeVisible();
  await expect(page.locator('#autoRefreshToggle')).toBeChecked();
});

test('stats grid populates from /api/v1/stats', async ({ page }) => {
  // Two static targets => two scheduled jobs; value leaves the "—" placeholder.
  await expect(page.locator('#totalJobs')).toHaveText('2', { timeout: 15000 });
  await expect(page.locator('#lastUpdate')).not.toHaveText(/—/);
});

test('job cards render, one up and one down', async ({ page }) => {
  await expect(page.locator('.vm-job-card')).toHaveCount(2, { timeout: 15000 });
  // States settle once probes have run at least once.
  await expect(page.locator('.vm-job-card--up')).toHaveCount(1, { timeout: 30000 });
  await expect(page.locator('.vm-job-card--down')).toHaveCount(1, { timeout: 30000 });
});

test('protocol distribution counts the two TCP probes', async ({ page }) => {
  await expect(page.locator('#protocolTCP .vm-protocol-card__count')).toHaveText('2', {
    timeout: 15000,
  });
});

test('targets table lists both targets', async ({ page }) => {
  await expect(page.locator('#targetsContent table.vm-table')).toBeVisible({ timeout: 15000 });
  await expect(page.locator('#targetsContent table.vm-table tbody tr')).toHaveCount(2);
});

test('status filter narrows the job list', async ({ page }) => {
  await expect(page.locator('.vm-job-card--down')).toHaveCount(1, { timeout: 30000 });

  await page.locator('.vm-filter-btn[data-filter="down"]').click();
  await expect(page.locator('.vm-filter-btn[data-filter="down"]')).toHaveClass(/active/);
  await expect(page.locator('.vm-job-card')).toHaveCount(1);

  await page.locator('.vm-filter-btn[data-filter="all"]').click();
  await expect(page.locator('.vm-job-card')).toHaveCount(2);
});

test('search filters jobs and shows empty state', async ({ page }) => {
  await expect(page.locator('.vm-job-card')).toHaveCount(2, { timeout: 15000 });

  await page.locator('#search').fill('127.0.0.1');
  await expect(page.locator('.vm-job-card')).toHaveCount(2);

  await page.locator('#search').fill('zzz-no-such-target');
  await expect(page.locator('.vm-empty')).toBeVisible();

  await page.locator('#search').fill('');
  await expect(page.locator('.vm-job-card')).toHaveCount(2);
});

test('health and API endpoints respond', async ({ page, request }) => {
  const health = await request.get('/health');
  expect(health.ok()).toBeTruthy();
  expect(await health.json()).toMatchObject({ status: 'healthy' });

  const ready = await request.get('/ready');
  expect(ready.ok()).toBeTruthy();
  expect(await ready.json()).toMatchObject({ ready: true });

  const stats = await request.get('/api/v1/stats');
  expect(stats.ok()).toBeTruthy();
  expect(await stats.json()).toHaveProperty('scheduler');

  const targets = await request.get('/api/v1/targets');
  expect(targets.ok()).toBeTruthy();
  expect(Array.isArray(await targets.json())).toBeTruthy();
});
