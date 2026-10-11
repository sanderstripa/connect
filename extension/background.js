const HOST='com.sanderstripa.connect';
let port,sequence=0,pending=new Map(),queue=Promise.resolve();
const snapshot={connected:false,nodes:[],activeNodeId:'',busy:false,operation:''};
async function publish(error=''){const saved=await chrome.storage.local.get(['desired','desiredNodeId']);await chrome.storage.local.set({view:{...snapshot,desired:!!saved.desired,desiredNodeId:saved.desiredNodeId||snapshot.activeNodeId,error}});chrome.action.setBadgeText({text:snapshot.connected?'ON':''});}
function request(action,data={}){if(!port){port=chrome.runtime.connectNative(HOST);const current=port;port.onMessage.addListener(m=>{Object.assign(snapshot,m);if(m.event==='stopped'){snapshot.connected=false;if(snapshot.busy)return;chrome.storage.local.get('desired').then(({desired})=>publish(desired?'Соединение прервано. Трафик заблокирован до переподключения.':''));return;}const p=pending.get(m.id);if(p){clearTimeout(p.timer);pending.delete(m.id);m.ok?p.resolve(m):p.reject(new Error(m.error||'Не удалось выполнить действие'));}});port.onDisconnect.addListener(()=>{const detail=chrome.runtime.lastError?.message;if(port===current)port=null;for(const p of pending.values()){clearTimeout(p.timer);p.reject(new Error(detail||'Помощник недоступен. Установите Connect с компонентом Chrome.'));}pending.clear();snapshot.connected=false;chrome.storage.local.get('desired').then(({desired})=>publish(desired?'Связь с помощником потеряна.':''));});}return new Promise((resolve,reject)=>{const id=++sequence;const timer=setTimeout(()=>{pending.delete(id);reject(new Error('Сервер не ответил. Проверьте настройки.'));},45000);pending.set(id,{resolve,reject,timer});port.postMessage({id,action,...data});});}
async function setProxy(){const settings=await chrome.proxy.settings.get({incognito:false});if(!['controllable_by_this_extension','controlled_by_this_extension'].includes(settings.levelOfControl))throw new Error('Настройки прокси заняты другим расширением или политикой Chrome.');await chrome.proxy.settings.set({scope:'regular',value:{mode:'fixed_servers',rules:{singleProxy:{scheme:'socks5',host:'127.0.0.1',port:17991},bypassList:['localhost','127.0.0.1','[::1]']}}});await chrome.privacy.network.webRTCIPHandlingPolicy.set({scope:'regular',value:'disable_non_proxied_udp'});}
async function disconnect(){await chrome.storage.local.set({desired:false,desiredNodeId:''});await chrome.proxy.settings.clear({scope:'regular'});await chrome.privacy.network.webRTCIPHandlingPolicy.clear({scope:'regular'});await request('disconnect');snapshot.connected=false;}
async function dispatch(m){
 snapshot.busy=true;snapshot.operation=m.action;await publish();
 let error='';
 try{
  if(m.action==='activate'){
   await request('status');
   if(!snapshot.nodes.some(n=>n.id===m.nodeId))throw new Error('Соединение не найдено.');
   if(!(snapshot.connected&&snapshot.activeNodeId===m.nodeId)){
    await setProxy();
    await chrome.storage.local.set({desired:true,desiredNodeId:m.nodeId});
    if(snapshot.connected)await request('disconnect');
    if(snapshot.activeNodeId!==m.nodeId)await request('select',{nodeId:m.nodeId});
    await request('connect');
   }
  }else if(m.action==='connect'){
   await request('status');if(!snapshot.nodes.length)throw new Error('Добавьте соединение.');
   await setProxy();await chrome.storage.local.set({desired:true,desiredNodeId:snapshot.activeNodeId});await request('connect');
  }else if(m.action==='disconnect')await disconnect();
  else if(m.action==='select'){
   const {desired}=await chrome.storage.local.get('desired');
   if(desired){await setProxy();await chrome.storage.local.set({desiredNodeId:m.nodeId});}
   await request('select',{nodeId:m.nodeId});if(desired)await request('connect');
  }else if(m.action==='import')await request('import',{link:m.link,name:m.name});
  else if(m.action==='delete'){
   const saved=await chrome.storage.local.get('desiredNodeId');
   if(snapshot.activeNodeId===m.nodeId||saved.desiredNodeId===m.nodeId)await disconnect();
   await request('delete',{nodeId:m.nodeId});
  }else if(m.action==='status'){
   await request('status');const saved=await chrome.storage.local.get(['desired','desiredNodeId']);
   if(saved.desired&&!snapshot.connected){
    if(saved.desiredNodeId&&!snapshot.nodes.some(n=>n.id===saved.desiredNodeId))await disconnect();
    else{await setProxy();if(saved.desiredNodeId&&snapshot.activeNodeId!==saved.desiredNodeId)await request('select',{nodeId:saved.desiredNodeId});await request('connect');}
   }
  }else throw new Error('Неизвестная команда');
 }catch(e){error=e.message;}
 snapshot.busy=false;snapshot.operation='';await publish(error);
 const saved=await chrome.storage.local.get(['desired','desiredNodeId']);
 return {...snapshot,ok:!error,desired:!!saved.desired,desiredNodeId:saved.desiredNodeId||snapshot.activeNodeId,error};
}
chrome.runtime.onMessage.addListener((m,sender,respond)=>{if(sender.id!==chrome.runtime.id)return;queue=queue.then(()=>dispatch(m));queue.then(respond);return true;});
chrome.alarms.onAlarm.addListener(()=>{queue=queue.then(async()=>{const {desired}=await chrome.storage.local.get('desired');if(desired)await dispatch({action:'status'});});});
chrome.runtime.onInstalled.addListener(()=>chrome.alarms.create('connect-health',{periodInMinutes:0.5}));
chrome.runtime.onStartup.addListener(()=>{queue=queue.then(()=>dispatch({action:'status'}));});
chrome.proxy.onProxyError.addListener(()=>{chrome.storage.local.get('desired').then(({desired})=>{if(desired){snapshot.connected=false;publish('Прокси недоступен. Трафик остаётся заблокированным.');}});});
