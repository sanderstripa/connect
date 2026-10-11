import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
const bindings = fs.readFileSync(path.resolve('bindings/github.com/SUDOKU-ASCII/sudoku-desktop/app.ts'), 'utf8');
const ids = Object.fromEntries([...bindings.matchAll(/export function (\w+)\([^]*?\$Call.ByID\((\d+)/g)].map(m => [Number(m[2]), m[1]]));

test('tray follows theme, toggles connection, selects profiles and allows capture keys', async ({page}) => {
  await page.setViewportSize({width: 220, height: 255});
  const cfg = {ui: {theme: 'light'}, activeNodeId: 'one', nodes: [{id:'one',name:'Первый'}, {id:'two',name:'Второй'}]};
  const state = {running:false};
  const calls: string[] = [];
  const errors: string[] = [];
  let requestedHeight = 0;
  page.on('pageerror', e => errors.push(e.message));
  await page.route('**/wails/runtime', async route => {
    const req = route.request().postDataJSON();
    let result: unknown = null;
    if(req.object === 6 && req.method === 33) requestedHeight = req.args.height;
    if(req.object === 0) {
      const method = ids[req.args.methodID];
      calls.push(method);
      if(method === 'GetConfig') result = cfg;
      if(method === 'GetState') result = state;
      if(method === 'StartProxy') { expect(req.args.args[0].withTun).toBe(true); state.running = true; }
      if(method === 'StopProxy') state.running = false;
      if(method === 'SwitchNode') cfg.activeNodeId = req.args.args[0];
    }
    await route.fulfill({contentType:'application/json',body: JSON.stringify(result)});
  });
  await page.goto('/?tray=1');
  await expect(page.getByText('Состояние: отключено')).toBeVisible();
  await expect.poll(() => requestedHeight).toBeGreaterThan(160);
  expect(requestedHeight).toBeLessThan(210);
  await page.setViewportSize({width:220,height:requestedHeight});
  await page.getByRole('button', {name:'Подключить',exact:true}).click();
  await expect(page.getByText('Состояние: подключено')).toBeVisible();
  await page.getByRole('button', {name:'Отключить',exact:true}).click();
  await page.getByRole('button', {name:'Выбор профиля'}).click();
  await page.getByRole('button', {name:'Второй'}).click();
  await expect(page.getByRole('button', {name:'Второй ✓'})).toBeVisible();
  await page.getByRole('button', {name:'Профили',exact:true}).click();
  for (const theme of ['light', 'dark']) {
    cfg.ui.theme = theme;
    await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
    await expect(page.locator('.tray-panel')).toHaveCSS('font-family', /Montserrat/);
    expect(await page.evaluate(() => document.documentElement.scrollHeight <= innerHeight)).toBe(true);
    if(process.env.SUDOKU_PREVIEW_DIR) await page.screenshot({path:path.join(process.env.SUDOKU_PREVIEW_DIR, `tray-${theme}.png`)});
  }
  await page.keyboard.press('PrintScreen');
  await expect(page.getByRole('button', {name:'Настройки'})).toBeVisible();
  expect(calls).toContain('SwitchNode');
  expect(errors).toEqual([]);
});
