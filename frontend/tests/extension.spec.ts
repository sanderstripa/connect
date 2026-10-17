import {test,expect,type Page} from '@playwright/test';
import fs from 'node:fs/promises';
import path from 'node:path';
async function setup(page:Page,{empty=false,failed=false}={}){
 const root=path.resolve('../extension');
 await page.route('**/chrome-preview/**',async route=>{
  const name=new URL(route.request().url()).pathname.replace('/chrome-preview/','');const file=path.resolve(root,name);
  if(!file.startsWith(root+path.sep))return route.fulfill({status:403});
  const type=name.endsWith('.css')?'text/css':name.endsWith('.js')?'text/javascript':name.endsWith('.ttf')?'font/ttf':name.endsWith('.png')?'image/png':'text/html';await route.fulfill({contentType:type,body:await fs.readFile(file)});
 });
 await page.addInitScript(({empty,failed})=>{
  const state:any={ok:true,connected:false,desired:false,busy:false,activeNodeId:'s',nodes:empty?[]:[{id:'x',name:'Amsterdam-Chrome',protocol:'xray'},{id:'s',name:'203.0.113.20',protocol:'sudoku'},{id:'m',name:'Мой Mist',protocol:'mist'}]};
  const listeners:any[]=[];(window as any).calls=[];
  (window as any).chrome={runtime:{async sendMessage(m:any){
   (window as any).calls.push(m);
   if(m.action!=='status')await new Promise(r=>setTimeout(r,60));
   state.error='';state.ok=true;
   if(m.action==='activate'){state.activeNodeId=m.nodeId;state.connected=!failed;state.desired=true;if(failed){state.ok=false;state.error='Не удалось подключить сервер';}}
   if(m.action==='disconnect'){state.connected=false;state.desired=false;}
   if(m.action==='delete'){state.nodes=state.nodes.filter((n:any)=>n.id!==m.nodeId);if(state.activeNodeId===m.nodeId){state.connected=false;state.desired=false;}}
   if(m.action==='import'){
    const p=m.link.startsWith('[Interface]')?'amneziawg':m.link.startsWith('sudoku://')?'sudoku':m.link.startsWith('vless://')?'xray':m.link.startsWith('{')?'mist':'';
    if(!p){state.ok=false;state.error='Некорректная ссылка';}else state.nodes.push({id:'new-'+state.nodes.length,name:m.name||(p==='sudoku'?'203.0.113.20':'Мой '+p),protocol:p});
   }
   const copy=JSON.parse(JSON.stringify(state));for(const fn of listeners)fn({view:{newValue:copy}},'local');return copy;
  },openOptionsPage(){throw Error('Unexpected settings tab');}},storage:{local:{async get(){return {preferences:JSON.parse(localStorage.getItem('preferences')||'null')};},async set(v:any){localStorage.setItem('preferences',JSON.stringify(v.preferences));}},onChanged:{addListener(fn:any){listeners.push(fn);}}}};
 },{empty,failed});
 await page.setViewportSize({width:320,height:560});await page.goto('/chrome-preview/popup.html');await page.evaluate(()=>document.fonts.ready);
}
async function assertFits(page:Page){expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);expect(await page.locator('body').evaluate(e=>e.getBoundingClientRect().height)).toBeLessThanOrEqual(560);}

test('compact popup switches profiles exclusively without a settings tab',async({page})=>{
 await setup(page);await expect(page.locator('.profile [role=switch]')).toHaveCount(3);await expect(page.locator('#power,#settings')).toHaveCount(0);
 const x=page.getByRole('switch',{name:'Xray — Amsterdam-Chrome'}),s=page.getByRole('switch',{name:'Sudoku — 203.0.113.20'});
 await x.click();await expect(s).toBeDisabled();await expect(x).toHaveAttribute('aria-checked','true');await expect(page.locator('.profile [role=switch][aria-checked=true]')).toHaveCount(1);
 await s.click();await expect(s).toHaveAttribute('aria-checked','true');await expect(x).toHaveAttribute('aria-checked','false');await expect(page.locator('.profile [role=switch][aria-checked=true]')).toHaveCount(1);
 expect(await page.evaluate(()=>(window as any).calls.filter((m:any)=>m.action==='activate').map((m:any)=>m.nodeId))).toEqual(['x','s']);
 for(const theme of ['light','dark'] as const){await page.emulateMedia({colorScheme:theme});await assertFits(page);if(process.env.SUDOKU_PREVIEW_DIR)await page.locator('body').screenshot({path:path.join(process.env.SUDOKU_PREVIEW_DIR,`chrome-list-${theme}.png`)});}
 await s.click();await expect(page.locator('.profile [role=switch][aria-checked=true]')).toHaveCount(0);await expect(page.getByText('Вы отключены')).toBeVisible();
 page.on('dialog',d=>d.accept());await page.getByRole('button',{name:'Удалить Мой Mist'}).click();await expect(page.locator('.profile [role=switch]')).toHaveCount(2);
});

test('empty popup imports links in place with automatic protocol detection',async({page})=>{
 await setup(page,{empty:true});await expect(page.locator('.profile [role=switch]')).toHaveCount(0);await expect(page.getByRole('button',{name:'Добавить',exact:true})).toBeVisible();
 const url=page.url();await page.getByRole('button',{name:'Добавить',exact:true}).click();await expect(page.getByRole('heading',{name:'Добавить соединение'})).toBeVisible();
 await page.getByLabel('Ссылка или конфигурация').fill('invalid');await page.getByRole('button',{name:'Добавить',exact:true}).click();await expect(page.getByRole('alert')).toHaveText('Некорректная ссылка');await expect(page.locator('#link')).toHaveValue('invalid');
 for(const [link,protocol,name]of [['sudoku://example','Sudoku','203.0.113.20'],['vless://example','Xray','Amsterdam-Chrome'],['{"token":"example"}','Mist','Мой Mist']]){
  if(await page.locator('#editor').isHidden())await page.getByRole('button',{name:'Добавить',exact:true}).click();
  await page.getByLabel('Ссылка или конфигурация').fill(link);await expect(page.locator('#detected')).toHaveText('Протокол · '+protocol);
  await page.locator('#name').fill(name);await assertFits(page);
  if(protocol==='Xray'&&process.env.SUDOKU_PREVIEW_DIR)await page.locator('body').screenshot({path:path.join(process.env.SUDOKU_PREVIEW_DIR,'chrome-add.png')});
  await page.getByRole('button',{name:'Добавить',exact:true}).click();await expect(page.getByRole('switch',{name:protocol+' — '+name})).toBeVisible();await expect(page.locator('#editor')).toBeHidden();expect(page.url()).toBe(url);
 }
 await expect(page.locator('.profile [role=switch]')).toHaveCount(3);await expect(page.locator('.profile [role=switch][aria-checked=true]')).toHaveCount(0);
});

test('failed connect stays visibly off and offers explicit release of blocked traffic',async({page})=>{
 await setup(page,{failed:true});await page.getByRole('switch',{name:'Xray — Amsterdam-Chrome'}).click();await expect(page.getByRole('alert')).toHaveText('Не удалось подключить сервер');await expect(page.locator('.profile [role=switch][aria-checked=true]')).toHaveCount(0);
 await expect(page.getByText('Трафик приостановлен')).toBeVisible();await page.getByRole('button',{name:'Отключить VPN'}).click();await expect(page.getByText('Вы отключены')).toBeVisible();await expect(page.locator('#release')).toBeHidden();
});

test('language and theme persist while inline editor keeps its draft and alignment',async({page})=>{
 await setup(page,{empty:true});await page.locator('#theme').click();const theme=await page.locator('html').getAttribute('data-theme');
 await page.locator('#language').click();await expect(page.locator('html')).toHaveAttribute('lang','en');await expect(page.locator('#add')).toContainText('Add');
 await page.locator('#add').click();await expect(page.getByRole('heading',{name:'Add connection'})).toBeVisible();await page.getByLabel('Link or configuration').fill('vless://example');
 await page.locator('#language').click();await expect(page.getByLabel('Ссылка или конфигурация')).toHaveValue('vless://example');
 const delta=await page.evaluate(()=>{const a=document.querySelector('#back')!.getBoundingClientRect(),b=document.querySelector('#editor-title')!.getBoundingClientRect();return Math.abs(a.y+a.height/2-b.y-b.height/2)});expect(delta).toBeLessThan(1);
 expect(await page.locator('#link').evaluate(e=>getComputedStyle(e).outlineStyle)).toBe('none');
 await page.locator('#language').click();await page.reload();await expect(page.locator('html')).toHaveAttribute('lang','en');await expect(page.locator('html')).toHaveAttribute('data-theme',theme!);await assertFits(page);
});

test('AmneziaWG file import detects protocol and creates an inline toggle',async({page})=>{
 await setup(page,{empty:true});await page.locator('#add').click();
 await page.locator('#config-file').setInputFiles({name:'AWG-test.conf',mimeType:'text/plain',buffer:Buffer.from('[Interface]\nAddress = 10.8.0.2/32\n[Peer]\nEndpoint = example.test:51820')});
 await expect(page.locator('#detected')).toContainText('AmneziaWG');await expect(page.locator('#name')).toHaveValue('AWG-test');await page.locator('#save').click();
 await expect(page.locator('#profiles')).toContainText('AmneziaWG');await page.getByRole('switch',{name:/AmneziaWG — AWG-test/}).click();await expect(page.getByRole('switch',{name:/AmneziaWG — AWG-test/})).toHaveAttribute('aria-checked','true');await assertFits(page);
});