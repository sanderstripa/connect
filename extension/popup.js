const $=id=>document.getElementById(id);
let state={nodes:[],connected:false},localBusy=false,localAction='',editing=false,localError='';
let lang='ru',light=matchMedia('(prefers-color-scheme: light)').matches;
const words={ru:{language:'Язык интерфейса',theme:'Светлая тема',add:'Добавить',title:'Добавить соединение',back:'Назад',link:'Ссылка или конфигурация',name:'Название · необязательно',server:'Мой сервер',placeholder:'sudoku://, vless://, JSON Mist или .conf AmneziaWG',protocol:'Протокол',footer:'Только Chrome',remove:'Удалить',confirm:'Удалить соединение',connecting:'Подключение…',disconnecting:'Отключение…',adding:'Добавление…',deleting:'Удаление…',connected:'Подключено',off:'Вы отключены',blocked:'Трафик приостановлен',disconnect:'Отключить VPN'},en:{language:'Interface language',theme:'Light theme',add:'Add',title:'Add connection',back:'Back',link:'Link or configuration',name:'Name · optional',server:'My server',placeholder:'sudoku://, vless://, Mist JSON or AmneziaWG .conf',protocol:'Protocol',footer:'Chrome only',remove:'Delete',confirm:'Delete connection',connecting:'Connecting…',disconnecting:'Disconnecting…',adding:'Adding…',deleting:'Deleting…',connected:'Connected',off:'Disconnected',blocked:'Traffic paused',disconnect:'Disconnect VPN'}};
const errors={'Соединение прервано. Трафик заблокирован до переподключения.':'Connection interrupted. Traffic is paused until reconnection.','Связь с помощником потеряна.':'Connection to the helper was lost.','Прокси недоступен. Трафик остаётся заблокированным.':'Proxy unavailable. Traffic remains paused.','Сервер не ответил. Проверьте настройки.':'The server did not respond. Check your connection.','Добавьте соединение.':'Add a connection.','Соединение не найдено.':'Connection not found.','Помощник недоступен. Установите Connect с компонентом Chrome.':'Helper unavailable. Install Connect with the Chrome component.','Настройки прокси заняты другим расширением или политикой Chrome.':'Proxy settings are controlled by another extension or Chrome policy.','Не удалось выполнить действие':'Could not complete the action','Не удалось подключить сервер':'Could not connect to the server','Некорректная ссылка':'Invalid link'};
const label=p=>({sudoku:'Sudoku',xray:'Xray',mist:'Mist',amneziawg:'AmneziaWG'}[p]||p||'Sudoku');
function detect(){const link=$('link').value.trim(),scheme=link.split(':')[0].toLowerCase();const p=(/^(?:\s*[#;][^\n]*\n)*\s*\[Interface\]/.test(link))||(scheme==='awg'||scheme==='vpn')?'AmneziaWG':['vless','vmess','trojan'].includes(scheme)?'Xray':['sudoku','http','https'].includes(scheme)?'Sudoku':scheme==='mist'||link.startsWith('{')?'Mist':'';$('detected').textContent=p?words[lang].protocol+' · '+p:'';}
function render(){
 const t=words[lang];$('file-label').textContent=lang==='en'?'Open .conf':'Открыть .conf';document.documentElement.lang=lang;document.documentElement.dataset.theme=light?'light':'dark';$('language').textContent=lang.toUpperCase();$('language').setAttribute('aria-label',t.language);$('theme').setAttribute('aria-label',t.theme);$('theme').setAttribute('aria-checked',String(light));$('editor-title').textContent=t.title;$('back').setAttribute('aria-label',t.back);$('link-label').textContent=t.link;$('name-label').textContent=t.name;$('name').placeholder=t.server;$('link').placeholder=t.placeholder;$('footer').textContent=t.footer;$('release').textContent=t.disconnect;$('add').replaceChildren();const plus=document.createElement('span');plus.textContent='+';plus.setAttribute('aria-hidden','true');$('add').append(plus,document.createTextNode(t.add));detect();
 const busy=localBusy||state.busy,profiles=$('profiles');profiles.replaceChildren();
 for(const n of state.nodes||[]){
  const checked=!busy&&!!state.connected&&n.id===state.activeNodeId;
  const row=document.createElement('div');row.className='profile';
  const identity=document.createElement('div');identity.className='identity';
  const protocol=document.createElement('span');protocol.className='protocol';protocol.textContent=label(n.protocol);
  const name=document.createElement('span');name.className='name';name.textContent=n.name||n.serverAddress||label(n.protocol);identity.append(protocol,name);
  const toggle=document.createElement('button');toggle.className='switch';toggle.type='button';toggle.setAttribute('role','switch');toggle.setAttribute('aria-checked',String(checked));toggle.setAttribute('aria-label',label(n.protocol)+' — '+name.textContent);toggle.disabled=!!busy;toggle.onclick=()=>act(checked?'disconnect':'activate',{nodeId:n.id});
  const remove=document.createElement('button');remove.className='remove';remove.type='button';remove.textContent='×';remove.setAttribute('aria-label',t.remove+' '+name.textContent);remove.disabled=!!busy;remove.onclick=()=>{if(confirm(t.confirm+' «'+name.textContent+'»?'))act('delete',{nodeId:n.id})};
  row.append(identity,toggle,remove);profiles.append(row);
 }
 $('connections').classList.toggle('empty',!state.nodes?.length);
 $('connections').hidden=editing;$('editor').hidden=!editing;
 $('status').hidden=!state.nodes?.length;
 $('status').textContent=busy?({disconnect:t.disconnecting,import:t.adding,delete:t.deleting}[localAction||state.operation]||t.connecting):state.connected?t.connected+' · '+label(state.nodes?.find(n=>n.id===state.activeNodeId)?.protocol):state.desired?t.blocked:t.off;
 $('add').disabled=!!busy;$('back').disabled=!!busy;$('save').disabled=!!busy;$('link').disabled=!!busy;$('name').disabled=!!busy;
 $('save').textContent=busy?t.adding:t.add;
 const error=localError||state.error||'';$('error').textContent=lang==='en'?(errors[error]||error):error;$('error').hidden=!error;
 $('release').hidden=!!busy||!state.desired||!!state.connected||editing;
}
async function act(action,data={}){
 if(localBusy||state.busy)return null;localBusy=true;localAction=action;localError='';state.error='';render();
 try{const result=await chrome.runtime.sendMessage({action,...data});state={...state,...result};return result}catch(e){localError=e.message;return null}finally{localBusy=false;localAction='';render()}
}
$('language').onclick=()=>{lang=lang==='ru'?'en':'ru';chrome.storage.local.set({preferences:{lang,light}});render()};
$('theme').onclick=()=>{light=!light;chrome.storage.local.set({preferences:{lang,light}});render()};
$('add').onclick=()=>{editing=true;localError='';state.error='';render();$('link').focus()};
$('back').onclick=()=>{editing=false;render()};
$('release').onclick=()=>act('disconnect');
$('link').oninput=detect;
$('import').onsubmit=async e=>{e.preventDefault();const link=$('link').value.trim();if(!link)return;const result=await act('import',{link,name:$('name').value.trim()});if(result?.ok){$('link').value='';$('name').value='';editing=false;render()}};
chrome.storage.onChanged.addListener((changes,area)=>{if(area==='local'&&changes.view){state={...state,...changes.view.newValue};render()}});
chrome.storage.local.get('preferences').then(({preferences:p})=>{if(p){lang=p.lang==='en'?'en':'ru';if(typeof p.light==='boolean')light=p.light;}render()});
chrome.runtime.sendMessage({action:'status'}).then(s=>{state=s;render()}).catch(e=>{localError=e.message;render()});
render();

$('config-file').onchange=async()=>{const file=$('config-file').files[0];if(!file)return;try{if(file.size>1048576)throw new Error(lang==='en'?'Configuration exceeds 1 MB':'Конфигурация больше 1 МБ');$('link').value=await file.text();if(!$('name').value)$('name').value=file.name.replace(/\.conf$/i,'');detect()}catch(e){localError=e.message;render()}};
