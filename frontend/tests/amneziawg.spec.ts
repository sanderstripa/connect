import {test,expect} from '@playwright/test';
import fs from 'node:fs';import path from 'node:path';
const bindings=fs.readFileSync(path.resolve('bindings/github.com/SUDOKU-ASCII/sudoku-desktop/app.ts'),'utf8');
const ids=Object.fromEntries([...bindings.matchAll(/export function (\w+)\([^]*?\$Call.ByID\((\d+)/g)].map(m=>[Number(m[2]),m[1]]));
test('desktop imports an AWG file and displays its saved profile',async({page})=>{
 const cfg:any={version:7,activeNodeId:'',nodes:[],routing:{proxyMode:'global'},tun:{enabled:true},core:{autoStart:false},ui:{theme:'dark',language:'ru'}};
 let imported='';
 await page.route('**/wails/runtime',async route=>{const req=route.request().postDataJSON();let result:any=null;if(req.object===0){const method=ids[req.args.methodID],args=req.args.args||[];
  if(method==='GetConfig')result=cfg;if(method==='GetState')result={running:false,traffic:{recentBandwidth:[]},recentLogs:[],kernel:{}};
  if(method==='ImportShortLink'){imported=args[1];const n={id:'awg',name:args[0],protocol:'amneziawg',serverAddress:'example.test:51820'};cfg.nodes.push(n);cfg.activeNodeId=n.id;result=n}
 }await route.fulfill({contentType:'application/json',body:JSON.stringify(result)})});
 await page.goto('/');await page.getByRole('button',{name:/Добавить соединение/}).click();
 const raw='[Interface]\nAddress = 10.8.0.2/32\n[Peer]\nEndpoint = example.test:51820';
 await page.locator('input[type=file]').setInputFiles({name:'Amnezia-test.conf',mimeType:'text/plain',buffer:Buffer.from(raw)});
 await expect(page.getByLabel('Ссылка соединения')).toHaveValue(raw);await expect(page.getByLabel('Название')).toHaveValue('Amnezia-test');
 await page.getByRole('button',{name:'Добавить соединение',exact:true}).click();await expect(page.locator('.server-card')).toContainText('Amnezia-test');expect(imported).toBe(raw);
 await page.locator('.server-card').click();await expect(page.getByText(/AMNEZIAWG · example.test:51820/)).toBeVisible();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
