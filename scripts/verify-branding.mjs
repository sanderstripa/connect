import fs from 'node:fs';import path from 'node:path';import assert from 'node:assert/strict';
const root=path.resolve(import.meta.dirname,'..'),read=p=>fs.readFileSync(path.join(root,p));
function png(b,size){assert.equal(b.subarray(1,4).toString(),'PNG');assert.equal(b.readUInt32BE(16),size);assert.equal(b.readUInt32BE(20),size)}
const manifest=JSON.parse(read('extension/manifest.json'));for(const [size,file]of Object.entries(manifest.icons)){png(read('extension/'+file),+size);assert.deepEqual(read('extension/'+file),read(`build/branding/icon-${size}.png`));assert.equal(manifest.action.default_icon[size],file)}
png(read('build/branding/source/app-round.png'),1024);
png(read('build/branding/source/tray-off.png'),1024);
png(read('build/branding/source/tray-on.png'),1024);
for(const [file,size]of [['build/appicon.png',1024],['cmd/installer/web/assets/icon.png',256],['frontend/src/assets/images/logo-universal.png',512],['extension/assets/icon.png',128]])assert.deepEqual(read(file),read(`build/branding/icon-${size}.png`));
const ico=read('build/windows/icon.ico');assert.equal(ico.readUInt16LE(2),1);assert.equal(ico.readUInt16LE(4),12);for(let i=0;i<ico.readUInt16LE(4);i++){const p=6+16*i,size=ico[p]||256,offset=ico.readUInt32LE(p+12),length=ico.readUInt32LE(p+8);const frame=ico.subarray(offset,offset+length);png(frame,size);assert.deepEqual(frame,read(`build/branding/icon-${size}.png`))}
assert.deepEqual(read('installer-core/products/vpn.ico'),ico);
assert.deepEqual(read('build/windows/installer.ico'),ico);
assert.deepEqual(read('installer-core/products/vpn.png'),read('build/branding/icon-256.png'));
for(const state of ['off','on']){
 const b=read(`build/tray/tray-${state}.ico`),sizes=[16,20,24,28,32,36,40,48,64,96,128,256];
 assert.equal(b.readUInt16LE(0),0);assert.equal(b.readUInt16LE(2),1);assert.equal(b.readUInt16LE(4),sizes.length);
 let end=6+16*sizes.length;
 for(const [i,size]of sizes.entries()){const p=6+16*i;assert.equal(b[p]||256,size);assert.equal(b[p+1]||256,size);assert.equal(b.readUInt16LE(p+6),32);assert.equal(b.readUInt32LE(p+12),end);const length=b.readUInt32LE(p+8);assert.deepEqual(b.subarray(end,end+length),read(`build/tray/${state}-${size}.png`));end+=length}
 assert.equal(end,b.length);
}
const icns=read('build/darwin/icons.icns');assert.equal(icns.subarray(0,4).toString(),'icns');assert.equal(icns.readUInt32BE(4),icns.length);let count=0;for(let offset=8;offset<icns.length;){const length=icns.readUInt32BE(offset+4);assert.ok(length>8&&offset+length<=icns.length);assert.equal(icns.subarray(offset+9,offset+12).toString(),'PNG');offset+=length;count++}assert.equal(count,7);
for(const size of [16,20,24,28,32,36,40,48,64]){png(read(`build/tray/off-${size}.png`),size);png(read(`build/tray/on-${size}.png`),size);assert.notDeepEqual(read(`build/tray/off-${size}.png`),read(`build/branding/icon-${size}.png`));assert.notDeepEqual(read(`build/tray/on-${size}.png`),read(`build/tray/off-${size}.png`))}
assert.equal(read('build/tray/logo.svg').toString(),read('build/branding/connect.svg').toString());assert.ok(read('build/tray/logo-monochrome.svg').toString().includes('currentColor'));
console.log('PASS: squircle app/setup/extension branding, transparent OFF/ON tray, Windows ICO and macOS ICNS match the approved assets');
