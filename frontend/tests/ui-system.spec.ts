import {test,expect} from '@playwright/test';
import fs from 'node:fs';import path from 'node:path';import crypto from 'node:crypto';
const bindings=fs.readFileSync(path.resolve('bindings/github.com/SUDOKU-ASCII/sudoku-desktop/app.ts'),'utf8');
const ids=Object.fromEntries([...bindings.matchAll(/export function (\w+)\([^]*?\$Call.ByID\((\d+)/g)].map(m=>[Number(m[2]),m[1]]));

test('original palette, font asset and native shell remain byte-identical',()=>{
 const expected:Record<string,string>={
  'src/style.css':'256b5dffe74b7d17da53f1de23bef6bd1b1418ed1f059d11bb801a12c6dda2bb',
  'src/assets/fonts/Montserrat.ttf':'0f7b311b2f3279e4eef9b2f968bcdbab6e28f4daeb1f049f4f278a902bcd82f7',
  '../main.go':'5165e72554327e65c3bc6244d859b88dea56e73999a3ad31b8a82b16234ebf27',
  '../docs/assets/CONNECT-settings-08.svg':'3f14c9094a96c13584a5514ef648a25cfbcb95371668036279ef4cb8051baf18'};
 // Normalize checked-out CRLF/LF text; the Montserrat binary is hashed as-is.
 for(const [file,hash] of Object.entries(expected)){
  const raw=fs.readFileSync(file);const data=file.endsWith('.ttf')?raw:Buffer.from(raw.toString('utf8').replace(/\r\n/g,'\n'));expect(crypto.createHash('sha256').update(data).digest('hex')).toBe(hash);
 }
});

test('unified controls retain original colours, settings-only preferences and keyboard focus',async({page})=>{
 const cfg:any={version:7,activeNodeId:'s',nodes:[{id:'s',name:'UI fixture',protocol:'sudoku',serverAddress:'example.test:443'}],routing:{proxyMode:'global'},tun:{enabled:true},core:{autoStart:false},ui:{theme:'light',language:'ru'}};
 await page.route('**/wails/runtime',async route=>{const req=route.request().postDataJSON();let result:any=null;if(req.object===0){const method=ids[req.args.methodID];if(method==='GetConfig')result=cfg;if(method==='SaveConfig')Object.assign(cfg,req.args.args[0]);if(method==='GetState')result={running:false,traffic:{recentBandwidth:[]},kernel:{},recentLogs:[]}}await route.fulfill({contentType:'application/json',body:JSON.stringify(result)})});
 await page.goto('/');await expect(page.locator('.titlebar button')).toHaveCount(2);await expect(page.locator('.titlebar')).toHaveCSS('height','54px');
 const footerOffset=await page.locator('footer').evaluate(e=>{const f=e.getBoundingClientRect(),s=e.querySelector('.footer-brand')!.getBoundingClientRect();return (s.top+s.height/2)-((f.top+parseFloat(getComputedStyle(e).borderTopWidth)+f.bottom)/2)});expect(footerOffset).toBeCloseTo(-2,0);
 const gear=page.locator('.settings-icon svg'),settings=page.getByRole('button',{name:'Настройки',exact:true});
 await expect(gear).toHaveAttribute('viewBox','5 0 85 85');await expect(gear).toHaveAttribute('width','24');await expect(gear).toHaveAttribute('height','24');
 const original=fs.readFileSync('../docs/assets/CONNECT-settings-08.svg','utf8');
 await expect(gear.locator('path')).toHaveAttribute('d',original.match(/ d="([^"]+)"/)![1]);await expect(gear.locator('circle')).toHaveAttribute('r','6.578');
 for(const theme of ['light','dark']){
  await page.locator('html').evaluate((e,t)=>e.setAttribute('data-theme',t),theme);
  await expect(settings).toHaveCSS('background-color','rgba(0, 0, 0, 0)');await settings.hover();await expect(settings).toHaveCSS('background-color','rgba(0, 0, 0, 0)');await expect(gear).toHaveCSS('background-color','rgba(0, 0, 0, 0)');
  for(const button of await page.locator('.titlebar button').all()){await button.hover();await expect(button).toHaveCSS('background-color','rgba(0, 0, 0, 0)')}
 }
 await page.locator('html').evaluate(e=>e.setAttribute('data-theme','light'));
 await page.getByRole('button',{name:'Настройки',exact:true}).click();await expect(page.locator('.titlebar button')).toHaveCount(2);
 await expect(page.getByRole('group',{name:'Язык интерфейса'})).toBeVisible();
 const primary=page.getByRole('button',{name:'Сохранить настройки',exact:true}),mode=page.getByRole('combobox',{name:'Режим'});
 await expect(primary).toHaveCSS('border-radius','8px');await expect(primary).toHaveCSS('min-height','36px');await expect(mode).toHaveCSS('font-size','13px');await expect(mode).toHaveCSS('border-radius','8px');
 expect(await page.locator('.switch').first().evaluate(e=>({w:e.getBoundingClientRect().width,h:e.getBoundingClientRect().height}))).toEqual({w:40,h:23});
 await mode.focus();await page.keyboard.press('Tab');await page.keyboard.press('Shift+Tab');await expect(mode).toBeFocused();await expect(mode).toHaveCSS('outline-width','2px');await page.keyboard.press('ArrowDown');await expect(page.getByRole('listbox')).toBeVisible();await page.keyboard.press('Escape');await expect(page.getByRole('listbox')).toHaveCount(0);
 for(const [button,bg,ink,fill,inverse] of [['Светлая','rgb(247, 247, 245)','rgb(23, 26, 29)','rgb(23, 26, 29)','rgb(255, 255, 255)'],['Тёмная','rgb(12, 12, 13)','rgb(237, 237, 238)','rgb(230, 231, 233)','rgb(18, 19, 21)']]){
  await page.getByRole('button',{name:button,exact:true}).click();await expect(page.locator('html')).toHaveCSS('background-color',bg);await expect(page.locator('html')).toHaveCSS('color',ink);await expect(primary).toHaveCSS('background-color',fill);await expect(primary).toHaveCSS('color',inverse);
 }
 await page.getByRole('button',{name:'EN',exact:true}).click();await expect(page.getByRole('heading',{name:'SETTINGS'})).toBeVisible();expect(cfg.ui.language).toBe('en');
 await page.emulateMedia({reducedMotion:'reduce'});await expect(page.getByRole('button',{name:'Save settings',exact:true})).toHaveCSS('transition-duration','0s');
});
