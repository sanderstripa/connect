import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';

const bindings = fs.readFileSync(path.resolve('bindings/github.com/SUDOKU-ASCII/sudoku-desktop/app.ts'), 'utf8');
const ids = Object.fromEntries([...bindings.matchAll(/export function (\w+)\([^]*?\$Call.ByID\((\d+)/g)].map(m => [Number(m[2]), m[1]]));

for (const failed of [false, true]) {
  test(`startup waits for initial data and reveals ${failed ? 'an actionable error' : 'the themed interface'} without waiting for version lookup`, async ({ page }) => {
    let releaseConfig!: () => void;
    const configGate = new Promise<void>(resolve => { releaseConfig = resolve; });
    let releaseVersion!: () => void;
    const versionGate = new Promise<void>(resolve => { releaseVersion = resolve; });
    const readiness: unknown[] = [];
    await page.route('**/wails/runtime', async route => {
      const req = route.request().postDataJSON();
      let result: unknown = null;
      if (req.object === 0) {
        const method = ids[req.args.methodID];
        if (method === 'GetConfig') {
          await configGate;
          if (failed) { await route.fulfill({ status: 503, body: 'Startup unavailable' }); return; }
          result = { version: 7, nodes: [], routing: { proxyMode: 'global' }, tun: { enabled: true }, core: {}, ui: { theme: 'dark', language: 'ru' } };
        }
        if (method === 'GetState') result = { running: false, traffic: { recentBandwidth: [] }, recentLogs: [] };
        if (req.args.methodName === 'main.App.GetUpdateStatus') {
          await versionGate;
          result = { installed: '0.4.3' };
        }
      }
      if (req.object === 3 && req.args.name === 'main:ready') {
        readiness.push({ data: req.args.data, loading: await page.locator('.loading').count(), theme: await page.locator('html').getAttribute('data-theme') });
      }
      await route.fulfill({ contentType: 'application/json', body: JSON.stringify(result) });
    });
    await page.goto('/');
    await expect(page.locator('.loading')).toBeVisible();
    expect(readiness).toEqual([]);
    releaseConfig();
    try {
      await expect.poll(() => readiness.length).toBe(1);
      await expect(page.locator('.loading')).toHaveCount(0);
      if (failed) await expect(page.getByRole('alert')).toContainText('Startup unavailable');
      else {
        expect(readiness).toEqual([{ data: { dark: true }, loading: 0, theme: 'dark' }]);
        await expect(page.getByRole('button', { name: 'Добавить соединение', exact: true })).toBeVisible();
      }
    } finally { releaseVersion(); }
  });
}
