const $=id=>document.getElementById(id);
let state={nodes:[],connected:false},localBusy=false,localAction='',editing=false,localError='';
const label=p=>({sudoku:'Sudoku',xray:'Xray',mist:'Mist'}[p]||p||'Sudoku');
function render(){
 const busy=localBusy||state.busy,profiles=$('profiles');profiles.replaceChildren();
 for(const n of state.nodes||[]){
  const checked=!busy&&!!state.connected&&n.id===state.activeNodeId;
  const row=document.createElement('div');row.className='profile';
  const identity=document.createElement('div');identity.className='identity';
  const protocol=document.createElement('span');protocol.className='protocol';protocol.textContent=label(n.protocol);
  const name=document.createElement('span');name.className='name';name.textContent=n.name||n.serverAddress||label(n.protocol);identity.append(protocol,name);
  const toggle=document.createElement('button');toggle.className='switch';toggle.type='button';toggle.setAttribute('role','switch');toggle.setAttribute('aria-checked',String(checked));toggle.setAttribute('aria-label',label(n.protocol)+' — '+name.textContent);toggle.disabled=!!busy;toggle.onclick=()=>act(checked?'disconnect':'activate',{nodeId:n.id});
  const remove=document.createElement('button');remove.className='remove';remove.type='button';remove.textContent='×';remove.setAttribute('aria-label','Удалить '+name.textContent);remove.disabled=!!busy;remove.onclick=()=>{if(confirm('Удалить соединение «'+name.textContent+'»?'))act('delete',{nodeId:n.id})};
  row.append(identity,toggle,remove);profiles.append(row);
 }
 $('connections').classList.toggle('empty',!state.nodes?.length);
 $('connections').hidden=editing;$('editor').hidden=!editing;
 $('status').hidden=!state.nodes?.length;
 $('status').textContent=busy?({disconnect:'Отключение…',import:'Добавление…',delete:'Удаление…'}[localAction||state.operation]||'Подключение…'):state.connected?'Подключено · '+label(state.nodes?.find(n=>n.id===state.activeNodeId)?.protocol):state.desired?'Соединение прервано · трафик приостановлен':'Вы отключены';
 $('add').disabled=!!busy;$('back').disabled=!!busy;$('save').disabled=!!busy;$('link').disabled=!!busy;$('name').disabled=!!busy;
 $('save').textContent=busy?'Добавление…':'Добавить';
 const error=localError||state.error||'';$('error').textContent=error;$('error').hidden=!error;
 $('release').hidden=!!busy||!state.desired||!!state.connected||editing;
}
async function act(action,data={}){
 if(localBusy||state.busy)return null;localBusy=true;localAction=action;localError='';state.error='';render();
 try{const result=await chrome.runtime.sendMessage({action,...data});state={...state,...result};return result}catch(e){localError=e.message;return null}finally{localBusy=false;localAction='';render()}
}
$('add').onclick=()=>{editing=true;localError='';state.error='';render();$('link').focus()};
$('back').onclick=()=>{editing=false;render()};
$('release').onclick=()=>act('disconnect');
$('link').oninput=()=>{const link=$('link').value.trim();const scheme=link.split(':')[0].toLowerCase();const p=['vless','vmess','trojan'].includes(scheme)?'Xray':['sudoku','http','https'].includes(scheme)?'Sudoku':scheme==='mist'||link.startsWith('{')?'Mist':'';$('detected').textContent=p?'Протокол · '+p:'';};
$('import').onsubmit=async e=>{e.preventDefault();const link=$('link').value.trim();if(!link)return;const result=await act('import',{link,name:$('name').value.trim()});if(result?.ok){$('link').value='';$('name').value='';$('detected').textContent='';editing=false;render()}};
chrome.storage.onChanged.addListener((changes,area)=>{if(area==='local'&&changes.view){state={...state,...changes.view.newValue};render()}});
chrome.runtime.sendMessage({action:'status'}).then(s=>{state=s;render()}).catch(e=>{localError=e.message;render()});
render();
