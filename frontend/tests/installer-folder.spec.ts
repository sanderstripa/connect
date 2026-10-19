import {test,expect} from '@playwright/test';
import fs from 'node:fs';import path from 'node:path';
test.use({viewport:{width:840,height:540}});
const root=path.resolve('../cmd/installer/web');
for(const language of ['ru','en'])test(`folder cancellation preserves path without warning (${language})`,async({page})=>{
 let fail=false,result={path:'',cancelled:true},installations=0;
 await page.route('**/*',async route=>{
  const p=new URL(route.request().url()).pathname;
  if(p==='/api/folder')return route.fulfill({status:fail?500:200,json:fail?{error:'access denied'}:result});
  if(p==='/api/install'){installations++;expect(route.request().postDataJSON().path).toBe('D:\\Selected\\Connect');return route.fulfill({json:{ok:true}})}
  if(p.startsWith('/api/'))return route.fulfill({json:{ok:true}});
  if(p==='/wails/runtime.js')return route.fulfill({contentType:'application/javascript',body:''});
  const file=path.resolve(root,p==='/'?'index.html':p.slice(1));if(!file.startsWith(root+path.sep))throw Error('asset');let body=fs.readFileSync(file);
  if(file.endsWith('index.html'))body=Buffer.from(body.toString().replace('__TOKEN__','test').replace('__MODE__','false').replace('__CONFIG__',JSON.stringify({path:'C:\\Program Files\\Connect',version:'0.2.5'})));
  await route.fulfill({body,contentType:({'.js':'application/javascript','.css':'text/css','.png':'image/png','.ttf':'font/ttf'}as any)[path.extname(file)]||'text/html'});
 });
 await page.goto('/');await page.locator('#settingsButton').click();await page.locator('#'+language).click();
 await page.locator('#browse').click();await expect(page.locator('#path')).toHaveValue('C:\\Program Files\\Connect');await expect(page.locator('#failure')).toBeHidden();
 await page.locator('#path').fill('E:\\My custom folder');await page.locator('#browse').click();await expect(page.locator('#path')).toHaveValue('E:\\My custom folder');await expect(page.locator('#failure')).toBeHidden();
 fail=true;await page.locator('#browse').click();await expect(page.locator('#error')).toBeVisible();await expect(page.locator('#path')).toHaveValue('E:\\My custom folder');
 fail=false;await page.locator('#retry').click();await expect(page.locator('#failure')).toBeHidden();await expect(page.locator('#path')).toHaveValue('E:\\My custom folder');
 result={path:'D:\\Selected\\Connect',cancelled:false};await page.locator('#browse').click();await expect(page.locator('#path')).toHaveValue(result.path);
 result={path:'',cancelled:true};await page.locator('#browse').click();await expect(page.locator('#path')).toHaveValue('D:\\Selected\\Connect');await expect(page.locator('#failure')).toBeHidden();await expect(page.locator('#accept')).toBeEnabled();
 expect(installations).toBe(0);await page.locator('#chrome').uncheck();await page.locator('#accept').click();await expect(page.locator('#finish')).toBeVisible();expect(installations).toBe(1);
});
