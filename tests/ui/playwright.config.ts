import { defineConfig, devices } from '@playwright/test';

// vmprober binary: Linux CI builds `bin/vmprober`; on Windows dev set
// VMPROBER_BIN=../../bin/vmprober.exe (or just start vmprober yourself — the
// running server is reused locally).
const vmproberBin = process.env.VMPROBER_BIN || '../../bin/vmprober';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI
    ? [['html', { open: 'never' }], ['list']]
    : [['list']],
  use: {
    baseURL: 'http://localhost:8429',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  webServer: {
    command: `${vmproberBin} --config vmprober.config.yaml`,
    url: 'http://localhost:8429/health',
    reuseExistingServer: !process.env.CI,
    timeout: 60 * 1000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
