const HOST='com.sanderstripa.connect';
let port,sequence=0,pending=new Map(),queue=Promise.resolve();
let closing=false,lifecycleEpoch=0;
let pendingHandoff=null;
const snapshot={connected:false,nodes:[],activeNodeId:'',busy:false,operation:'',desktopRunning:false,confirmDesktop:false};
async function publish(error=''){const saved=await chrome.storage.session.get(['desired','desiredNodeId']);await chrome.storage.local.set({view:{...snapshot,desired:!!saved.desired,desiredNodeId:saved.desiredNodeId||snapshot.activeNodeId,error}});chrome.action.setBadgeText({text:snapshot.connected?'ON':''});}
function request(action,data={}){if(closing)return Promise.reject(new Error('Browser session ended'));if(!port){port=chrome.runtime.connectNative(HOST);const current=port;port.onMessage.addListener(m=>{if(port!==current||closing)return;
if(m.event==='release-browser'){releaseToDesktop(current,m).catch(console.error);return;}
if(m.event==='desktop-status'){snapshot.desktopRunning=m.desktopRunning;publish().catch(console.error);return;}
Object.assign(snapshot,m);if(m.event==='stopped'){snapshot.connected=false;if(snapshot.busy)return;chrome.storage.session.get('desired').then(({desired})=>publish(desired?'Соединение прервано. Трафик заблокирован до переподключения.':''));return;}const p=pending.get(m.id);if(p){clearTimeout(p.timer);pending.delete(m.id);m.ok?p.resolve(m):p.reject(new Error(m.error||'Не удалось выполнить действие'));}});port.onDisconnect.addListener(()=>{if(port!==current)return;const detail=chrome.runtime.lastError?.message;if(port===current)port=null;for(const p of pending.values()){clearTimeout(p.timer);p.reject(new Error(detail||'Помощник недоступен. Установите Connect с компонентом Chrome.'));}pending.clear();snapshot.connected=false;snapshot.desktopRunning=false;chrome.storage.session.get('desired').then(({desired})=>publish(desired?'Связь с помощником потеряна.':''));});}return new Promise((resolve,reject)=>{const id=++sequence;const timer=setTimeout(()=>{pending.delete(id);reject(new Error('Сервер не ответил. Проверьте настройки.'));},110000);pending.set(id,{resolve,reject,timer});port.postMessage({id,action,...data});});}
async function setProxy(){const settings=await chrome.proxy.settings.get({incognito:false});if(!['controllable_by_this_extension','controlled_by_this_extension'].includes(settings.levelOfControl))throw new Error('Настройки прокси заняты другим расширением или политикой Chrome.');const {bypassDomains=[]}=await chrome.storage.local.get('bypassDomains');await chrome.proxy.settings.set({scope:'regular',value:{mode:'fixed_servers',rules:{singleProxy:{scheme:'socks5',host:'127.0.0.1',port:17991},bypassList:['localhost','127.0.0.1','[::1]',...bypassDomains.flatMap(d=>[d,'*.'+d])]}}});await chrome.privacy.network.webRTCIPHandlingPolicy.set({scope:'regular',value:'disable_non_proxied_udp'});}
async function disconnect(){await chrome.storage.session.set({desired:false,desiredNodeId:''});await chrome.proxy.settings.clear({scope:'regular'});await chrome.privacy.network.webRTCIPHandlingPolicy.clear({scope:'regular'});await request('disconnect');snapshot.connected=false;}
async function openPopup(popup='popup.html'){
 try{await chrome.action.setPopup({popup});await chrome.action.openPopup();}
 finally{await chrome.action.setPopup({popup:''});}
}
async function releaseToDesktop(current,message){
 let handoffError='';
 try{
  await chrome.privacy.network.webRTCIPHandlingPolicy.clear({scope:'regular'});
  await chrome.storage.session.set({desired:false,desiredNodeId:''});
  await chrome.proxy.settings.clear({scope:'regular'});
  snapshot.connected=false;snapshot.desktopRunning=true;
 }catch(e){
  handoffError=e.message;
  await chrome.storage.session.set({desired:true,desiredNodeId:message.activeNodeId}).catch(console.error);
  await setProxy().catch(console.error);
 }
 if(port===current)current.postMessage({action:'handoff-ack',handoffId:message.handoffId,handoffError});
 await publish(handoffError);
}
async function connectNode(nodeId,confirmed=false){
 if(!nodeId)throw new Error('Добавьте соединение.');
 if(snapshot.desktopRunning&&!confirmed){
  pendingHandoff=nodeId;snapshot.confirmDesktop=true;
  return;
 }
 await setProxy();
 await chrome.storage.session.set({desired:true,desiredNodeId:nodeId});
 if(snapshot.connected&&snapshot.activeNodeId!==nodeId)await request('disconnect');
 if(snapshot.activeNodeId!==nodeId)await request('select',{nodeId});
 try{await request('connect',{confirmed});}
 catch(e){
  if(e.message==='CONNECT_DESKTOP_VPN_ACTIVE'){
   await chrome.storage.session.set({desired:false,desiredNodeId:''});
   await clearProxy();
   pendingHandoff=nodeId;snapshot.confirmDesktop=true;
   return;
  }
  throw e;
 }
}
async function dispatch(m){
 await initialized;const epoch=lifecycleEpoch;
 if(closing)return {...snapshot,ok:true,desired:false};
 snapshot.busy=true;snapshot.operation=m.action;await publish();
 let error='';
 try{
  if(m.action==='toggle'){
   const {desired}=await chrome.storage.session.get('desired');
   if(desired)await disconnect();
   else{
    await request('status');
    const {defaultNodeId}=await chrome.storage.local.get('defaultNodeId');
    const nodeId=snapshot.nodes.some(n=>n.id===defaultNodeId)?defaultNodeId:snapshot.activeNodeId||snapshot.nodes[0]?.id;
    await connectNode(nodeId);
   }
  }else if(m.action==='cancel-handoff'){
   pendingHandoff=null;snapshot.confirmDesktop=false;
  }else if(m.action==='confirm-handoff'){
   const nodeId=pendingHandoff;
   pendingHandoff=null;snapshot.confirmDesktop=false;
   if(nodeId){await request('status');await connectNode(nodeId,true);}
  }else if(m.action==='default'){
   await request('status');
   if(!snapshot.nodes.some(n=>n.id===m.nodeId))throw new Error('Соединение не найдено.');
   await chrome.storage.local.set({defaultNodeId:m.nodeId});
  }else if(m.action==='routing'){
   const bypassDomains=normalizeDomains(m.domains);
   const previous=await chrome.storage.local.get('bypassDomains');
   await chrome.storage.local.set({bypassDomains});
   const {desired}=await chrome.storage.session.get('desired');
   if(desired){try{await setProxy()}catch(e){await chrome.storage.local.set({bypassDomains:previous.bypassDomains||[]});throw e;}}
  }else if(m.action==='activate'){
   await request('status');
   if(!snapshot.nodes.some(n=>n.id===m.nodeId))throw new Error('Соединение не найдено.');
   if(!(snapshot.connected&&snapshot.activeNodeId===m.nodeId))await connectNode(m.nodeId);
  }else if(m.action==='connect'){
   await request('status');await connectNode(snapshot.activeNodeId);
  }else if(m.action==='disconnect')await disconnect();
  else if(m.action==='select'){
   const {desired}=await chrome.storage.session.get('desired');
   if(desired){await setProxy();await chrome.storage.session.set({desiredNodeId:m.nodeId});}
   await request('select',{nodeId:m.nodeId});if(desired)await request('connect');
  }else if(m.action==='import')await request('import',{link:m.link,name:m.name});
  else if(m.action==='delete'){
   const saved=await chrome.storage.session.get('desiredNodeId');
   if(snapshot.activeNodeId===m.nodeId||saved.desiredNodeId===m.nodeId)await disconnect();
   await request('delete',{nodeId:m.nodeId});
   const {defaultNodeId}=await chrome.storage.local.get('defaultNodeId');if(defaultNodeId===m.nodeId)await chrome.storage.local.remove(['defaultNodeId']);
  }else if(m.action==='status'){
   await request('status');const saved=await chrome.storage.session.get(['desired','desiredNodeId']);
   if(saved.desired&&!snapshot.connected&&!snapshot.desktopRunning){
    if(saved.desiredNodeId&&!snapshot.nodes.some(n=>n.id===saved.desiredNodeId))await disconnect();
    else{await setProxy();if(saved.desiredNodeId&&snapshot.activeNodeId!==saved.desiredNodeId)await request('select',{nodeId:saved.desiredNodeId});await request('connect');}
   }
  }else throw new Error('Неизвестная команда');
 }catch(e){error=e.message;}
 if(closing||epoch!==lifecycleEpoch){await releaseSession();error='';}
 snapshot.busy=false;snapshot.operation='';await publish(error);
 const saved=await chrome.storage.session.get(['desired','desiredNodeId']);
 return {...snapshot,ok:!error,desired:!!saved.desired,desiredNodeId:saved.desiredNodeId||snapshot.activeNodeId,error};
}

async function clearProxy(){await chrome.proxy.settings.clear({scope:'regular'});await chrome.privacy.network.webRTCIPHandlingPolicy.clear({scope:'regular'});}
function closeNativePort(){const old=port;port=null;for(const p of pending.values()){clearTimeout(p.timer);p.reject(new Error('Browser session ended'));}pending.clear();old?.disconnect();}
async function releaseSession(){
 await chrome.storage.session.set({desired:false,desiredNodeId:''});
 closeNativePort();snapshot.connected=false;
 await clearProxy();await publish();
}
// Session storage survives worker suspension, but never a browser restart.
const initialized=(async()=>{
 const {initialized}=await chrome.storage.session.get('initialized');
 if(!initialized){await chrome.storage.session.set({initialized:true,desired:false,desiredNodeId:''});await clearProxy();snapshot.connected=false;await publish();}
 await chrome.storage.local.remove(['desired','desiredNodeId']);
 await chrome.action.setPopup({popup:''});
})();
async function endBrowserSession(){
 const windows=await chrome.windows.getAll({windowTypes:['normal','popup']});
 if(windows.length)return;
 closing=true;lifecycleEpoch++;await initialized;await releaseSession();
}
chrome.windows.onRemoved.addListener(()=>endBrowserSession().catch(console.error));
chrome.windows.onCreated.addListener(w=>{if(['normal','popup'].includes(w.type))closing=false;});
chrome.runtime.onMessage.addListener((m,sender,respond)=>{if(sender.id!==chrome.runtime.id)return;queue=queue.catch(()=>{}).then(()=>dispatch(m));queue.then(respond,e=>respond({ok:false,error:e.message,connected:false}));return true;});
chrome.alarms.onAlarm.addListener(()=>{queue=queue.catch(()=>{}).then(async()=>{await initialized;await endBrowserSession();const {desired}=await chrome.storage.session.get('desired');if(desired&&!closing)await dispatch({action:'status'});});});
chrome.runtime.onInstalled.addListener(()=>chrome.alarms.create('connect-health',{periodInMinutes:0.5}));
chrome.runtime.onStartup.addListener(()=>{queue=queue.catch(()=>{}).then(async()=>{await initialized;await publish();});});
chrome.proxy.onProxyError.addListener(()=>{chrome.storage.session.get('desired').then(({desired})=>{if(desired&&!closing){snapshot.connected=false;publish('Прокси недоступен. Трафик остаётся заблокированным.');}});});

// Only domain names are accepted: no proxy bypass syntax, ports or path rules.
function normalizeDomains(input){
 if(typeof input!=='string'||input.length>65536)throw new Error('Введите список доменов (до 64 КБ).');
 const result=[];
 for(let entry of input.split(/[\n,;]+/).map(s=>s.trim()).filter(Boolean)){
  entry=entry.replace(/^\*\./,'');
  let url;try{url=new URL(entry.includes('://')?entry:'https://'+entry)}catch{throw new Error('Некорректный домен: '+entry)}
  const domain=url.hostname.toLowerCase().replace(/\.$/,'');
  if(!['http:','https:'].includes(url.protocol)||url.username||url.password||url.port||url.search||url.hash||url.pathname!=='/'||domain.length>253||!domain.split('.').every(part=>/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(part)))throw new Error('Некорректный домен: '+entry);
  if(!result.includes(domain))result.push(domain);
 }
 return result;
}
function enqueue(message){queue=queue.catch(()=>{}).then(()=>dispatch(message));return queue;}
chrome.action.onClicked.addListener(()=>enqueue({action:'toggle'}).then(result=>{if(result.confirmDesktop)return openPopup('popup.html#handoff');}).catch(console.error));
chrome.contextMenus.onClicked.addListener(info=>{
 if(info.menuItemId!=='connect-settings')return;
 // Set a popup only while opening settings, so ordinary left clicks keep toggling.
 openPopup().catch(async e=>{console.error(e);await publish('Не удалось открыть настройки. Повторите правый клик.');});
});
chrome.runtime.onInstalled.addListener(()=>{
 chrome.contextMenus.removeAll(()=>chrome.contextMenus.create({id:'connect-settings',title:'Настройки / Settings',contexts:['action']}));
});
