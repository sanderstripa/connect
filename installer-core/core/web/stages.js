// Optional stages are selected by declared capabilities, never product-name checks.
window.setupStages={
 hotkey:{
  vk:44,mod:0,relogin:false,listening:false,
  async start(){const key=await call('hotkey-current');this.vk=key.vk;this.mod=key.mod;stage='hotkey';},
  render(){
   const en=lang==='en', tr=(ru,english)=>en?english:ru;
   const labels={hotkeyTitle:tr('Горячая клавиша','Keyboard shortcut'),hotkeyHint:tr('Нажмите поле, затем клавишу для создания скриншота.','Click the field, then press a key to capture screenshots.'),hotkeyNext:tr('Далее','Next'),confirmTitle:tr('Заменить системное действие?','Replace the Windows action?'),confirmHint:tr('Назначить Print Screen приложению Screenshot вместо стандартного инструмента Windows?','Assign Print Screen to Screenshot instead of the default Windows capture tool?'),hotkeyCancel:tr('Отмена','Cancel'),hotkeyReplace:tr('Да, заменить','Yes, replace')};
   for(const [id,label]of Object.entries(labels))$(id).textContent=label;
   const names={44:'Print Screen',13:'Enter',32:'Space',27:'Escape',8:'Backspace',9:'Tab'};
   const key=names[this.vk]||(this.vk>=112&&this.vk<=135?'F'+(this.vk-111):String.fromCharCode(this.vk));
   $('keyField').textContent=this.listening?tr('Нажмите клавишу…','Press a key…'):[this.mod&2?'Ctrl':'',this.mod&4?'Shift':'',this.mod&1?'Alt':'',key].filter(Boolean).join(' + ');
   $('hotkeyRelogin').hidden=stage!=='finish'||!this.relogin;
   $('hotkeyRelogin').textContent=tr('Для активации Print Screen может потребоваться повторный вход в Windows.','Sign out and back in to activate Print Screen if needed.');
  },
  async save(replace){
   if(busy)return;busy=true;clearError();render();try{const r=await call('hotkey-save',{vk:this.vk,mod:this.mod,replace});this.relogin=r.relogin;stage='finish';this.listening=false}catch(e){error(e.message)}finally{busy=false;render()}
  },
  init(){
   $('keyField').onclick=()=>{this.listening=true;render();$('keyField').focus()};
   const choose=e=>{if(!this.listening)return;if(['Shift','Control','Alt','Meta'].includes(e.key))return;if(e.key==='PrintScreen'&&e.type==='keydown')return;e.preventDefault();this.vk=e.keyCode;this.mod=(e.ctrlKey?2:0)|(e.shiftKey?4:0)|(e.altKey?1:0);this.listening=false;render()};
   $('keyField').addEventListener('keydown',choose);$('keyField').addEventListener('keyup',e=>{if(e.key==='PrintScreen')choose(e)});
   $('hotkeyNext').onclick=()=>{if(this.vk===44&&this.mod===0){stage='confirm';render()}else this.save(false)};
   $('hotkeyCancel').onclick=()=>{stage='hotkey';render()};$('hotkeyReplace').onclick=()=>this.save(true);
  }
 }
};
