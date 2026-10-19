import {test,expect} from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import {execFileSync} from 'node:child_process';
const root=path.resolve('../cmd/installer/web');
const cache=process.env.GOMODCACHE||execFileSync(process.env.CONNECT_GO||'go',['env','GOMODCACHE'],{encoding:'utf8'}).trim();
const runtime=fs.readFileSync(path.join(cache,'github.com/wailsapp/wails/v3@v3.0.0-alpha.74/internal/assetserver/bundledassets/runtime.js'));
for(const uninstall of [false,true])test(`native drag messages and interactive controls (${uninstall?'uninstall':'install'})`,async({page})=>{
 await page.addInitScript(()=>{
  const w=window as any;
  w.dragMessages=[];
  w._wails={environment:{OS:'windows'},flags:{}};
  Object.defineProperty(w,'chrome',{value:{webview:{postMessage:(msg:any)=>w.dragMessages.push(msg)}},configurable:true});
 });
 await page.route('**/*',async route=>{
  const url=new URL(route.request().url());
  if(url.pathname==='/wails/runtime.js')return route.fulfill({body:runtime,contentType:'application/javascript'});
  if(url.pathname==='/wails/custom.js'||url.pathname==='/wails/transport.js')return route.fulfill({body:'',contentType:'application/javascript'});
  if(url.pathname==='/wails/runtime')return route.fulfill({json:{OS:'windows'}});
  if(url.pathname.startsWith('/api/'))return route.fulfill({json:{ok:true}});
  const file=path.resolve(root,url.pathname==='/'?'index.html':url.pathname.slice(1));
  if(!file.startsWith(root+path.sep))throw Error('invalid asset');
  let body=fs.readFileSync(file);
  if(file.endsWith('index.html'))body=Buffer.from(body.toString().replace('__TOKEN__','test').replace('__MODE__',String(uninstall)).replace('__CONFIG__',JSON.stringify({path:'C:\\Program Files\\Connect',version:'test'})));
  await route.fulfill({body,contentType:({'.js':'application/javascript','.css':'text/css','.png':'image/png','.ttf':'font/ttf'}as any)[path.extname(file)]||'text/html'});
 });
 await page.goto('/');
 await expect.poll(()=>page.evaluate(()=>!!(window as any).wails)).toBe(true);
 await expect.poll(()=>page.evaluate(()=>(window as any).dragMessages.includes('wails:runtime:ready'))).toBe(true);
 const count=()=>page.evaluate(()=>(window as any).dragMessages.filter((x:any)=>x==='wails:drag').length);
 const gesture=async(x:number,y:number)=>{await page.mouse.move(x,y);await page.mouse.down();await page.mouse.move(x+6,y+4);await page.mouse.up()};
 for(const [x,y]of [[20,35],[160,35],[400,450],[420,510],[600,220]]){const before=await count();await gesture(x,y);await expect.poll(count).toBe(before+1)}
 // The actual Wails runtime must leave normal controls clickable.
 await page.locator('#settingsButton').click();
 await expect(page.locator('#settings')).toBeVisible();
 const before=await count();await page.locator('#en').click();await page.locator('#dark').click();
 if(!uninstall){await page.locator('#desktop').uncheck();await page.locator('#startMenu').uncheck();await page.locator('#chrome').uncheck();await page.locator('#path').fill('D:\\CONNECT')}
 expect(await count()).toBe(before);
 await page.locator('#back').click();
 const after=await count();await page.locator('#primary').click();await expect(page.locator('#finish')).toBeVisible();
 if(!uninstall)await page.locator('#launch').uncheck();
 expect(await count()).toBe(after);
});
