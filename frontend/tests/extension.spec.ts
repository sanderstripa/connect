import {test,expect} from '@playwright/test';
import fs from 'node:fs/promises';import path from 'node:path';
test('Chrome popup has one connection control and keeps protocols inside settings',async({page})=>{
 const root=path.resolve('../extension');
 await page.route('**/chrome-preview/**',async route=>{const name=new URL(route.request().url()).pathname.replace('/chrome-preview/','');if(name.includes('..'))return route.fulfill({status:403});const type=name.endsWith('.css')?'text/css':name.endsWith('.js')?'text/javascript':name.endsWith('.ttf')?'font/ttf':name.endsWith('.png')?'image/png':'text/html';await route.fulfill({contentType:type,body:await fs.readFile(path.join(root,name))});});
 await page.addInitScript(()=>{
  const state:any={ok:true,connected:false,activeNodeId:'x',nodes:[{id:'x',name:'Мой Xray',protocol:'xray'},{id:'s',name:'Мой Sudoku',protocol:'sudoku'},{id:'m',name:'Мой Mist',protocol:'mist'}]};
  (window as any).chrome={runtime:{async sendMessage(m:any){if(m.action==='connect')state.connected=true;if(m.action==='disconnect')state.connected=false;if(m.action==='select')state.activeNodeId=m.nodeId;return {...state};},openOptionsPage(){(window as any).optionsOpened=true;}},storage:{onChanged:{addListener(){}}}};
 });
 await page.setViewportSize({width:320,height:420});await page.goto('/chrome-preview/popup.html');
 const power=page.getByRole('button',{name:'Подключить VPN',exact:true});await expect(power).toBeEnabled();await expect(page.locator('button')).toHaveCount(2);await expect(page.locator('nav')).toHaveCount(0);await power.click();await expect(page.getByRole('button',{name:'Отключить VPN',exact:true})).toHaveAttribute('aria-pressed','true');await expect(page.getByText('Соединение активно')).toBeVisible();await page.getByRole('button',{name:'Отключить VPN',exact:true}).click();await expect(page.getByText('Вы отключены')).toBeVisible();
 if(process.env.SUDOKU_PREVIEW_DIR)await page.screenshot({path:path.join(process.env.SUDOKU_PREVIEW_DIR,'chrome-popup.png')});
 await page.getByRole('button',{name:'Настройки',exact:true}).click();expect(await page.evaluate(()=>(window as any).optionsOpened)).toBe(true);
 await page.setViewportSize({width:850,height:780});await page.goto('/chrome-preview/settings.html');await expect(page.locator('.profile')).toHaveCount(3);await page.getByRole('button',{name:'Mist',exact:true}).click();await expect(page.locator('.profile')).toHaveCount(1);await expect(page.locator('.profile')).toContainText('✓ Мой Mist');
});
