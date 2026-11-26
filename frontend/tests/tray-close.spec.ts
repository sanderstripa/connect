import { test, expect } from '@playwright/test';
import fs from 'node:fs';
const ids = Object.fromEntries([...fs.readFileSync('bindings/github.com/SUDOKU-ASCII/sudoku-desktop/app.ts','utf8')
  .matchAll(/export function (\w+)\([^]*?\$Call.ByID\((\d+)/g)].map(m => [Number(m[2]), m[1]]));

test('closing to tray returns each section to home and keeps live VPN and pending settings', async ({page}) => {
  let config:any = {version:7, activeNodeId:'x', nodes:[{id:'x', name:'Loopback profile', protocol:'xray', serverAddress:'127.0.0.1:8443'}],routing:{proxyMode:'pac',customRules:'',customRulesEnabled:true,ruleUrls:[]},tun:{enabled:true},core:{autoStart:false},ui:{theme:'light',language:'ru'}};
  const calls:string[] = [];
  await page.route('**/wails/runtime', async route => {
    const req = route.request().postDataJSON();
    const method = req.args?.methodName?.split('.').pop() || ids[req.args?.methodID];
    let result:any = null;
    if (req.object === 0) {
      calls.push(method);
      if (method === 'GetConfig') result = config;
      if (method === 'GetState') result = {running:true,coreRunning:true,tunRunning:true,traffic:{recentBandwidth:[]},recentLogs:[],kernel:{}};
      if (method === 'SaveConfigForNextConnection') config = req.args.args[0];
      if (method === 'GetUpdateStatus') result={id:req.args.args[0],installed:'v1',previous:'',releases:[],source:'',error:''};
    }
    await route.fulfill({contentType:'application/json',body:JSON.stringify(result)});
  });
  const emit = (name:string, data?:string) => page.evaluate(({name,data}) => (window as any)._wails.dispatchWailsEvent({name,data}), {name,data});
  const settings = () => page.getByRole('button',{name:'Настройки',exact:true}).click();
  const close = () => page.getByRole('button',{name:'Скрыть в трей',exact:true}).click();
  const home = async () => {
    await expect(page.locator('main.page-home')).toBeVisible();
    await expect(page.locator('main:not(.page-home)')).toHaveCount(0);
    await expect(page.getByText('Соединение активно')).toBeVisible();
  };
  await page.goto('/');
  await home();
  await settings();
  await page.locator('main textarea').fill('loopback.test');
  await close();
  await home();
  await expect.poll(() => config.routing.customRules).toContain('loopback.test');
  await settings();
  await page.getByRole('button',{name:'Обновление',exact:true}).click();
  await emit('main:hide-to-tray');
  await home();
  await settings();
  await page.getByRole('button',{name:'Журнал подключения',exact:true}).click();
  await close();
  await home();
  await emit('main:navigate','servers');
  await expect(page.locator('main.page-servers')).toBeVisible();
  await close();
  await home();
  await emit('main:navigate','settings');
  await expect(page.locator('main.page-settings')).toBeVisible();
  await page.getByRole('button',{name:'Свернуть',exact:true}).click();
  await expect(page.locator('main.page-settings')).toBeVisible();
  expect(calls.filter(name => ['StopProxy','StartProxy','RestartProxy'].includes(name))).toEqual([]);
});
