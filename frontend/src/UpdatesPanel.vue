<script setup lang="ts">
import {computed,onMounted,ref} from 'vue';
import {Call} from '@wailsio/runtime';
type Release={version:string;prerelease:boolean;published:string};
type Component={id:string;installed:string;build?:string;previous:string;releases:Release[];error:string;source:string};
const props=defineProps<{language:string;running:boolean}>();
const en=computed(()=>props.language==='en');
const text=(ru:string,eng:string)=>en.value?eng:ru;
const ids=['app','xray','geo','sudoku','mist','amneziawg'];
const items=ref<Record<string,Component>>({}),selected=ref<Record<string,string>>({}),working=ref(''),notice=ref(''),problem=ref('');
const previews=ref(localStorage.getItem('connect-update-preview')!=='false');
const name=(id:string)=>({app:text('Приложение','Application'),xray:'Xray',geo:'GeoIP / GeoSite',sudoku:'Sudoku',mist:'Mist',amneziawg:'AmneziaWG'}[id]||id);
const sameVersion=(id:string)=>!!selected.value[id]&&selected.value[id].replace(/^v/,'')===items.value[id]?.installed.replace(/^v/,'');
const call=(method:string,...args:unknown[])=>Call.ByName('main.App.'+method,...args);
async function status(id:string,check:boolean){const item=await call('GetUpdateStatus',id,check,previews.value) as Component;items.value[id]=item;selected.value[id]=item.releases?.[0]?.version||'';}
async function perform(key:string,fn:()=>Promise<unknown>){if(working.value)return;working.value=key;problem.value='';notice.value='';try{await fn()}catch(e){problem.value=e instanceof Error?e.message:String(e)}finally{working.value=''}}
async function check(id:string){await perform(id,()=>status(id,true))}
async function checkAll(){await perform('all',async()=>{for(const id of ids)await status(id,true)})}
function previewChanged(){localStorage.setItem('connect-update-preview',String(previews.value));for(const item of Object.values(items.value))item.releases=[];selected.value={};}
const previousApp=computed(()=>{const c=items.value.app;if(c?.previous)return c.previous;if(!c?.releases)return '';const current=c.installed.replace(/^v/,'');return c.releases.find(r=>r.version.replace(/^v/,'')!==current&&compareVersion(r.version,c.installed)<0)?.version||''});
function compareVersion(a:string,b:string){const x=a.replace(/^v/,'').split(/[.+-]/).map(Number),y=b.replace(/^v/,'').split(/[.+-]/).map(Number);for(let i=0;i<3;i++){const d=(x[i]||0)-(y[i]||0);if(d)return d}return 0;}
async function install(id:string,version:string){
 if(!version)return;
 await perform(id,async()=>{await call('UpdateComponent',id,version);if(id==='app'){notice.value=text('Запуск установщика…','Launching setup…');return}await status(id,false);notice.value=text('Установлено. Переподключите VPN, чтобы использовать новую версию.','Installed. Reconnect VPN to use the new version.')});
}
async function rollback(id:string){
 if(id==='app'&&!items.value.app?.previous){await install(id,previousApp.value);return}
 const version=items.value[id]?.previous;if(!version)return;
 await perform(id,async()=>{await call('RollbackComponent',id);if(id==='app'){notice.value=text('Запуск установщика…','Launching setup…');return}await status(id,false);notice.value=text('Предыдущая версия восстановлена. Переподключите VPN.','Previous version restored. Reconnect VPN.')});
}
onMounted(()=>perform('loading',async()=>{for(const id of ids)await status(id,false)}));
</script>
<template>
 <section class="updates-panel">
  <div class="updates-intro"><p class="hint">{{text('Установленные версии и доступные выпуски. Ядра общие для приложения и расширения.','Installed versions and available releases. Components are shared by the app and extension.')}}</p>
  <label class="toggle-row">{{text('Предварительные выпуски','Pre-release versions')}}<input type="checkbox" role="switch" v-model="previews" :disabled="!!working" @change="previewChanged"><span class="switch"/></label>
  <button class="text-row" :disabled="!!working" @click="checkAll">{{working==='all'?text('Проверка…','Checking…'):text('Проверить всё','Check all')}}</button></div>
  <article v-for="id in ids" :key="id" class="update-card" :aria-label="name(id)">
   <div class="update-heading"><h2>{{name(id)}}</h2><button :disabled="!!working" @click="check(id)">{{working===id?text('Подождите…','Please wait…'):text('Проверить','Check')}}</button></div>
   <p class="update-installed">{{text('Установлено','Installed')}} <strong>{{items[id]?.installed||text('Не установлено','Not installed')}}</strong></p>
   <p v-if="items[id]?.build" class="update-source">{{items[id].build}}</p>
   <p v-if="items[id]?.source" class="update-source"><a :href="'https://github.com/'+items[id].source" target="_blank" rel="noopener noreferrer">{{text('Источник обновлений','Update source')}}</a></p>
   <p v-if="id==='mist'" class="hint">{{text('Экспериментальный Mist. Совместимые сборки Connect с адаптером SOCKS.','Experimental Mist. Connect-compatible builds with the SOCKS adapter.')}}</p>
   <label v-if="items[id]?.releases?.length" class="release-label">{{text('Доступные версии','Available versions')}}
    <select v-model="selected[id]" :disabled="!!working" :aria-label="name(id)+' '+text('версия','version')"><option v-for="release in items[id].releases" :key="release.version" :value="release.version">{{release.version}}{{release.prerelease?text(' · предварительная',' · pre-release'):''}}</option></select>
   </label>
   <p v-if="items[id]?.error" class="hint update-error" role="status">{{items[id].error}}</p>
   <div class="update-actions">
    <button :disabled="!!working||!selected[id]||sameVersion(id)||(id==='app'&&running)" @click="install(id,selected[id])">{{sameVersion(id)?text('Уже установлено','Already installed'):id==='app'?text('Установить и закрыть Connect','Install and close Connect'):text('Установить','Install')}}</button>
    <button :disabled="!!working||!(id==='app'?previousApp:items[id]?.previous)||(id==='app'&&running)" @click="rollback(id)">{{text('Откат','Roll back')}}<span v-if="id==='app'?previousApp:items[id]?.previous"> · {{id==='app'?previousApp:items[id].previous}}</span></button>
   </div>
   <p v-if="id==='app'&&running" class="hint">{{text('Для обновления приложения отключите VPN.','Disconnect VPN to update the application.')}}</p>
  </article>
  <p v-if="notice" class="hint" role="status">{{notice}}</p><p v-if="problem" class="notice error" role="alert">{{problem}}</p>
 </section>
</template>
<style scoped>
.updates-panel{padding-bottom:12px}.updates-intro{margin-bottom:18px}.updates-intro .toggle-row{font-size:11px;gap:12px}.updates-intro .text-row{font-size:11px}.update-card{border-top:1px solid var(--line);padding:16px 0}.update-heading{display:flex;align-items:center;justify-content:space-between;gap:10px}.update-heading h2{font-size:13px;font-weight:500;margin:0}.update-heading button{font-size:10px;color:var(--muted);padding:5px}.update-installed{font-size:10px;color:var(--muted);margin:9px 0 5px;overflow-wrap:anywhere}.update-installed strong{display:inline;font-weight:500;color:var(--text);margin-left:5px}.update-source a{color:inherit;text-decoration:none}.update-source a:hover{text-decoration:underline}.update-source{font-size:9px;color:var(--muted);margin:0 0 8px;overflow-wrap:anywhere}.release-label{display:block;font-size:10px;color:var(--muted);margin-top:12px}.release-label select{display:block;width:100%;height:36px;margin-top:7px;padding:0 10px;border:1px solid var(--line);border-radius:8px;background:var(--bg);color:var(--text);font:inherit}.update-actions{display:flex;flex-wrap:wrap;gap:8px;margin-top:12px}.update-actions button{padding:8px 10px;background:var(--hover);border-radius:7px;font-size:10px}.update-actions button:disabled{opacity:.4;cursor:default}.update-card .hint{font-size:10px;line-height:1.6;margin:8px 0}.update-error{overflow-wrap:anywhere}
</style>
