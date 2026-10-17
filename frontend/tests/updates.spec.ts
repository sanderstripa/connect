import {test,expect} from '@playwright/test';
import fs from 'node:fs';import path from 'node:path';
const bindings=fs.readFileSync(path.resolve('bindings/github.com/SUDOKU-ASCII/sudoku-desktop/app.ts'),'utf8');
const ids=Object.fromEntries([...bindings.matchAll(/export function (\w+)\([^]*?\$Call.ByID\((\d+)/g)].map(m=>[Number(m[2]),m[1]]));
test('updates show installed versions, preview releases, per-component updates and rollback',async({page})=>{
 let cfg:any={version:7,activeNodeId:'',nodes:[],routing:{proxyMode:'global',customRules:''},tun:{enabled:true},core:{autoStart:false},ui:{theme:'light',language:'ru'}};
 const versions:any={app:'0.1.6',xray:'v26.3.27',sudoku:'v0.5.1',mist:'v0.1.6',geo:'',amneziawg:'v0.1.8'};const previous:any={};const calls:any[]=[];
 const catalog:any={amneziawg:[{version:'v0.1.8',prerelease:false}],app:[{version:'v0.1.6',prerelease:false},{version:'v0.1.5',prerelease:false}],xray:[{version:'v26.10.10',prerelease:true},{version:'v26.8.1',prerelease:false}],sudoku:[{version:'v0.5.1',prerelease:false}],geo:[{version:'20261017',prerelease:false}],mist:[{version:'v0.1.6',prerelease:false}]};
 page.on('dialog',()=>{throw new Error('Updates must start without a confirmation dialog')});
 await page.route('**/wails/runtime',async route=>{const req=route.request().postDataJSON();let result:any=null;if(req.object===0){const method=req.args.methodName?.split('.').pop()||ids[req.args.methodID];const args=req.args.args||[];
 if(method==='GetConfig')result=cfg;if(method==='GetState')result={running:false,traffic:{recentBandwidth:[]},recentLogs:[],kernel:{}};
 if(method==='SaveConfig')cfg=args[0];
 if(method==='GetUpdateStatus'){const [id,check,preview]=args;calls.push({method,id,check,preview});result={id,installed:versions[id],previous:previous[id]||'',source:'official/repository',releases:check?catalog[id].filter((r:any)=>preview||!r.prerelease):[],error:''}}
 if(method==='UpdateComponent'){const [id,version]=args;calls.push({method,id,version});previous[id]=versions[id];versions[id]=version;result={version}}
 if(method==='RollbackComponent'){const [id]=args;calls.push({method,id});[versions[id],previous[id]]=[previous[id],versions[id]];result={version:versions[id]}}
 }await route.fulfill({contentType:'application/json',body:JSON.stringify(result)})});
 await page.goto('/');await page.getByRole('button',{name:'Настройки',exact:true}).click();await page.getByRole('button',{name:'Обновление',exact:true}).click();
 await expect(page.getByRole('heading',{name:'ОБНОВЛЕНИЕ',exact:true})).toBeVisible();const x=page.getByRole('article',{name:'Xray',exact:true});await expect(x).toContainText('v26.3.27');await x.getByRole('button',{name:'Проверить',exact:true}).click();await expect(x.getByRole('combobox')).toContainText('v26.10.10 · предварительная');await x.getByRole('button',{name:'Установить',exact:true}).click();await expect(x).toContainText('v26.10.10');await x.getByRole('button',{name:/Откат/}).click();await expect(x.locator('.update-installed')).toContainText('v26.3.27');
 await page.getByRole('switch',{name:'Предварительные выпуски'}).uncheck();await x.getByRole('button',{name:'Проверить',exact:true}).click();await expect(x.getByRole('combobox')).not.toContainText('v26.10.10');
 await page.getByRole('button',{name:'Проверить всё',exact:true}).click();await expect(page.getByRole('article',{name:'GeoIP / GeoSite'}).getByRole('combobox')).toContainText('20261017');await expect(page.getByRole('article',{name:'Приложение',exact:true}).getByRole('button',{name:/Откат/})).toBeEnabled();
 if(process.env.SUDOKU_PREVIEW_DIR){await page.screenshot({path:path.join(process.env.SUDOKU_PREVIEW_DIR,'updates-light.png')});await page.locator('html').evaluate(e=>e.setAttribute('data-theme','dark'));await expect(page.locator('.update-actions button').first()).toHaveCSS('background-color','rgb(26, 27, 30)');await page.screenshot({path:path.join(process.env.SUDOKU_PREVIEW_DIR,'updates-dark.png')});}
 expect(calls.some(c=>c.method==='UpdateComponent'&&c.id==='xray')).toBe(true);expect(calls.some(c=>c.method==='RollbackComponent'&&c.id==='xray')).toBe(true);
 await page.getByRole('button',{name:'Язык интерфейса'}).click();await expect(page.getByRole('heading',{name:'UPDATES',exact:true})).toBeVisible();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
