import fs from 'node:fs';import path from 'node:path';
const root=path.resolve(import.meta.dirname,'..'),read=n=>fs.readFileSync(path.join(root,`build/branding/icon-${n}.png`));
const sizes=[16,20,24,28,32,36,40,48,64,96,128,256],header=Buffer.alloc(6+16*sizes.length);header.writeUInt16LE(1,2);header.writeUInt16LE(sizes.length,4);let offset=header.length;const frames=[];
for(const [i,size]of sizes.entries()){const b=read(size),p=6+16*i;header[p]=header[p+1]=size===256?0:size;header.writeUInt16LE(1,p+4);header.writeUInt16LE(32,p+6);header.writeUInt32LE(b.length,p+8);header.writeUInt32LE(offset,p+12);offset+=b.length;frames.push(b)}
fs.writeFileSync(path.join(root,'build/windows/icon.ico'),Buffer.concat([header,...frames]));
const blocks=[];for(const [type,size]of [['icp4',16],['icp5',32],['icp6',64],['ic07',128],['ic08',256],['ic09',512],['ic10',1024]]){const b=read(size),h=Buffer.alloc(8);h.write(type);h.writeUInt32BE(b.length+8,4);blocks.push(h,b)}const h=Buffer.alloc(8);h.write('icns');h.writeUInt32BE(8+blocks.reduce((n,b)=>n+b.length,0),4);fs.writeFileSync(path.join(root,'build/darwin/icons.icns'),Buffer.concat([h,...blocks]));
