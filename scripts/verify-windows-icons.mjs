import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
const root=path.resolve(import.meta.dirname,'..');
const ico=fs.readFileSync(path.join(root,'build/windows/icon.ico'));
const expected=new Map();
for(let i=0;i<ico.readUInt16LE(4);i++){
 const p=6+16*i;
 expected.set(ico[p]||256,ico.subarray(ico.readUInt32LE(p+12),ico.readUInt32LE(p+12)+ico.readUInt32LE(p+8)));
}
for(const file of ['Connect.exe','Connect-Setup.exe']){
 const b=fs.readFileSync(path.join(root,'build/bin',file)),pe=b.readUInt32LE(0x3c);
 assert.equal(b.toString('ascii',pe,pe+4),'PE\0\0');
 const optional=pe+24,magic=b.readUInt16LE(optional),sectionCount=b.readUInt16LE(pe+6),sections=optional+b.readUInt16LE(pe+20);
 const resourceRVA=b.readUInt32LE(optional+(magic===0x20b?112:96)+16);
 function offset(rva){for(let i=0;i<sectionCount;i++){const p=sections+i*40,start=b.readUInt32LE(p+12),length=Math.max(b.readUInt32LE(p+8),b.readUInt32LE(p+16));if(rva>=start&&rva<start+length)return b.readUInt32LE(p+20)+rva-start}throw Error('RVA outside PE sections')}
 const base=offset(resourceRVA),resources=new Map();
 function walk(relative,keys=[]){const p=base+relative,count=b.readUInt16LE(p+12)+b.readUInt16LE(p+14);for(let i=0;i<count;i++){const entry=p+16+i*8,id=b.readUInt32LE(entry),target=b.readUInt32LE(entry+4),next=[...keys,id];if(target&0x80000000)walk(target&0x7fffffff,next);else{const data=base+target,start=offset(b.readUInt32LE(data)),length=b.readUInt32LE(data+4);resources.set(next.join('/'),b.subarray(start,start+length))}}}
 walk(0);
 const groups=[...resources].filter(([key])=>key.startsWith('14/'));
 assert.ok(groups.length,`${file}: no RT_GROUP_ICON`);
 for(const [key,group]of groups){
  assert.equal(group.readUInt16LE(4),expected.size,`${file}: frame count`);
  const language=key.split('/')[2],seen=new Set();
  for(let i=0;i<group.readUInt16LE(4);i++){const p=6+14*i,size=group[p]||256,id=group.readUInt16LE(p+12);seen.add(size);assert.deepEqual(resources.get(`3/${id}/${language}`),expected.get(size),`${file}: compiled ${size}px icon differs`)}
  assert.equal(seen.size,expected.size);
 }
 console.log(`PASS: ${file} compiled Windows icon groups match approved ICO at all ${expected.size} sizes`);
}
