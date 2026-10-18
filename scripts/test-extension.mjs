import { test } from 'node:test';
import assert from 'node:assert/strict';
import vm from 'node:vm';
import fs from 'node:fs';
function harness({failedConnect=false,failedNode='',delay=0,cold=false,savedDesired=false}={}){
 const storage={initialized:!cold,desired:savedDesired},local={desired:true,preferences:{theme:'dark'}},changes=[],commands=[],listeners={};
 let connected=false,activeNodeId='one',nodes=[{id:'one',name:'Xray',protocol:'xray'},{id:'two',name:'Sudoku',protocol:'sudoku'}];
 const windows=[{id:1,type:'normal'}];
 const chrome={windows:{async getAll(){return [...windows]},onRemoved:{addListener(fn){listeners.removed=fn}},onCreated:{addListener(fn){listeners.created=fn}}},runtime:{id:'connect-test',lastError:null,connectNative(){
  const p={disconnect(){p.closed=true;connected=false;p.drop?.()},onMessage:{addListener(fn){p.receive=fn}},onDisconnect:{addListener(fn){p.drop=fn}},postMessage(m){
   commands.push(m.action);
   setTimeout(()=>{
    if(p.closed)return;
    if(m.action==='select')activeNodeId=m.nodeId;
    if(m.action==='disconnect')connected=false;
    if(m.action==='delete'){nodes=nodes.filter(n=>n.id!==m.nodeId);if(activeNodeId===m.nodeId)activeNodeId=nodes[0]?.id||'';}
    const failed=m.action==='connect'&&(failedConnect||activeNodeId===failedNode);
    if(m.action==='connect')connected=!failed;
    p.receive({id:m.id,ok:!failed,error:failed?'offline':'',connected,activeNodeId,port:17991,nodes});
   },delay);
  }};listeners.port=p;return p;
 },onMessage:{addListener(fn){listeners.message=fn}},onInstalled:{addListener(){}},onStartup:{addListener(fn){listeners.startup=fn}}},
 storage:{session:{async get(k){return typeof k==='string'?{[k]:storage[k]}:storage},async set(v){Object.assign(storage,v)}},local:{async get(k){return typeof k==='string'?{[k]:local[k]}:local},async set(v){Object.assign(local,v);if(v.view)storage.view=v.view},async remove(keys){for(const k of keys)delete local[k]}}},
 action:{setBadgeText(){}},proxy:{settings:{async get(){return {levelOfControl:'controllable_by_this_extension'}},async set(v){changes.push(['proxy',v])},async clear(v){changes.push(['clear',v])}},onProxyError:{addListener(fn){listeners.proxyError=fn}}},
 privacy:{network:{webRTCIPHandlingPolicy:{async set(v){changes.push(['webrtc',v])},async clear(v){changes.push(['rtcClear',v])}}}},
 alarms:{onAlarm:{addListener(fn){listeners.alarm=fn}},create(){}}};
 vm.runInContext(fs.readFileSync(new URL('../extension/background.js',import.meta.url),'utf8'),vm.createContext({chrome,setTimeout,clearTimeout,console}));
 return {storage,local,windows,changes,commands,listeners,send:m=>new Promise(resolve=>listeners.message(typeof m==='string'?{action:m}:m,{id:chrome.runtime.id},resolve))};
}

test('connect scopes proxy to Chrome and disconnect releases both settings',async()=>{
 const h=harness();const result=await h.send('connect');assert.equal(result.connected,true);assert.equal(h.storage.desired,true);
 assert.equal(h.changes[0][0],'proxy');assert.equal(h.changes[0][1].value.rules.singleProxy.port,17991);assert.equal(h.changes[0][1].scope,'regular');
 await h.send('disconnect');assert.equal(h.storage.desired,false);assert.equal(h.changes.at(-2)[0],'clear');assert.equal(h.changes.at(-1)[0],'rtcClear');assert.deepEqual(h.commands,['status','connect','disconnect']);
});
test('failed connection keeps blocking proxy until explicit disconnect',async()=>{
 const h=harness({failedConnect:true});const result=await h.send('connect');assert.equal(result.ok,false);assert.equal(result.connected,false);assert.equal(h.storage.desired,true);assert.equal(h.changes.filter(v=>v[0]==='clear').length,0);
 await h.send('disconnect');assert.equal(h.storage.desired,false);
});
test('unexpected native disconnect reports failure without releasing proxy',async()=>{
 const h=harness();await h.send('connect');h.listeners.port.drop();await new Promise(r=>setTimeout(r,10));assert.equal(h.storage.view.connected,false);assert.equal(h.storage.desired,true);assert.equal(h.changes.filter(v=>v[0]==='clear').length,0);
});
test('activate starts chosen profile directly and switches exclusively without releasing proxy',async()=>{
 const h=harness();let r=await h.send({action:'activate',nodeId:'two'});assert.equal(r.activeNodeId,'two');assert.equal(r.connected,true);
 r=await h.send({action:'activate',nodeId:'one'});assert.equal(r.activeNodeId,'one');assert.equal(r.connected,true);assert.equal(h.storage.desiredNodeId,'one');
 assert.deepEqual(h.commands,['status','select','connect','status','disconnect','select','connect']);assert.equal(h.changes.filter(v=>v[0]==='clear').length,0);
 const count=h.commands.length;await h.send({action:'activate',nodeId:'one'});assert.equal(h.commands.length,count+1,'already active profile only reads status');
});
test('failed switch shows no connected profile and remembers requested profile for retry',async()=>{
 const h=harness({failedNode:'two'});await h.send({action:'activate',nodeId:'one'});const r=await h.send({action:'activate',nodeId:'two'});
 assert.equal(r.ok,false);assert.equal(r.connected,false);assert.equal(r.desired,true);assert.equal(h.storage.desiredNodeId,'two');assert.equal(h.storage.view.busy,false);assert.equal(h.changes.filter(v=>v[0]==='clear').length,0);
 await h.send('disconnect');assert.equal(h.storage.desired,false);
});
test('deleting active profile disables desired connection and prevents automatic switch to another',async()=>{
 const h=harness();await h.send({action:'activate',nodeId:'two'});const r=await h.send({action:'delete',nodeId:'two'});
 assert.equal(r.connected,false);assert.equal(r.desired,false);assert.equal(r.nodes.length,1);assert.equal(h.storage.desiredNodeId,'');assert.deepEqual(h.commands.slice(-2),['disconnect','delete']);
});
test('queued toggles retain exclusive selection and publish busy state for a reopened popup',async()=>{
 const h=harness({delay:10});const first=h.send({action:'activate',nodeId:'two'});await new Promise(r=>setTimeout(r,3));assert.equal(h.storage.view.busy,true);
 const second=h.send({action:'activate',nodeId:'one'});await first;const r=await second;assert.equal(r.activeNodeId,'one');assert.equal(r.connected,true);assert.equal(h.storage.view.busy,false);
 assert.deepEqual(h.commands,['status','select','connect','status','disconnect','select','connect']);
});

test('intentional disconnect ignores delayed stopped and proxy error events',async()=>{
 const h=harness();await h.send('connect');await h.send('disconnect');
 h.listeners.port.receive({event:'stopped',connected:false});h.listeners.proxyError();
 await new Promise(r=>setTimeout(r,10));assert.equal(h.storage.view.error,'');assert.equal(h.storage.view.desired,false);
 h.listeners.port.drop();await new Promise(r=>setTimeout(r,10));assert.equal(h.storage.view.error,'');
});

const settle=()=>new Promise(r=>setTimeout(r,30));
test('browser restart clears legacy intent and proxy without reconnecting or losing preferences',async()=>{
 const h=harness({cold:true,savedDesired:true});h.listeners.startup();await settle();await h.send('status');h.listeners.alarm();await settle();
 assert.equal(h.storage.desired,false);assert.equal(h.local.desired,undefined);assert.equal(h.local.preferences.theme,'dark');assert.ok(h.changes.some(c=>c[0]==='clear'));assert.ok(!h.commands.includes('connect'));
});
test('worker restart in the same browser session retains explicit connection intent',async()=>{
 const h=harness({savedDesired:true});await h.send('status');assert.equal(h.storage.desired,true);assert.ok(h.commands.includes('connect'));assert.equal(h.changes.filter(c=>c[0]==='clear').length,0);
});
test('only closing the last window disconnects; reopening does not reconnect',async()=>{
 const h=harness();await h.send('connect');h.windows.push({id:2,type:'normal'});h.windows.pop();await h.listeners.removed();await settle();assert.equal(h.storage.desired,true);
 h.windows.length=0;await h.listeners.removed();await settle();assert.equal(h.storage.desired,false);assert.equal(h.listeners.port.closed,true);assert.equal(h.storage.view.connected,false);assert.ok(h.changes.some(c=>c[0]==='clear'));
 const count=h.commands.filter(c=>c==='connect').length;h.windows.push({id:3,type:'normal'});h.listeners.created(h.windows[0]);await h.send('status');h.listeners.alarm();await settle();assert.equal(h.commands.filter(c=>c==='connect').length,count);assert.equal(h.storage.desired,false);
});
test('closing during an in-flight activation cannot restore proxy or desired connection',async()=>{
 const h=harness({delay:30});const result=h.send('connect');await new Promise(r=>setTimeout(r,40));h.windows.length=0;await h.listeners.removed();await result;await settle();
 assert.equal(h.storage.desired,false);assert.equal(h.storage.view.connected,false);assert.equal(h.changes.at(-2)[0],'clear');assert.equal(h.changes.at(-1)[0],'rtcClear');assert.equal(h.listeners.port.closed,true);
});
