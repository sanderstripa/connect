from pathlib import Path
import ctypes,hashlib,json,mmap,struct

review=Path(__file__).resolve().parents[2]
root=review/'source'
k=ctypes.WinDLL('kernel32',use_last_error=True)
k.LoadLibraryExW.argtypes=[ctypes.c_wchar_p,ctypes.c_void_p,ctypes.c_uint];k.LoadLibraryExW.restype=ctypes.c_void_p
k.FindResourceW.argtypes=[ctypes.c_void_p,ctypes.c_void_p,ctypes.c_void_p];k.FindResourceW.restype=ctypes.c_void_p
k.LoadResource.argtypes=[ctypes.c_void_p,ctypes.c_void_p];k.LoadResource.restype=ctypes.c_void_p
k.LockResource.argtypes=[ctypes.c_void_p];k.LockResource.restype=ctypes.c_void_p
k.SizeofResource.argtypes=[ctypes.c_void_p,ctypes.c_void_p];k.SizeofResource.restype=ctypes.c_ulong
k.FreeLibrary.argtypes=[ctypes.c_void_p]
products={'Connect':('vpn','payload/Connect.exe'),'Screenshot':('screenshot','Screenshot.exe'),'Screencast':('screencast','payload.zip'),'Torrent':('torrent','payload.exe'),'Soulu':('soulu','payload.zip')}
results=[]
for name,(adapter,payload) in products.items():
 exe=review/f'{name}-Setup-x64.exe'
 with exe.open('rb') as f,mmap.mmap(f.fileno(),0,access=mmap.ACCESS_READ) as binary:
  pe=struct.unpack_from('<I',binary,0x3c)[0];assert binary[pe:pe+4]==b'PE\0\0';assert struct.unpack_from('<H',binary,pe+4)[0]==0x8664
  checks=[]
  for file in [root/'adapters'/adapter/payload,root/'core/web/assets/Montserrat.ttf',root/('core/web/assets/mobius.png' if adapter=='vpn' else f'adapters/{adapter}/ribbon.png')]:
   size=file.stat().st_size
   with file.open('rb') as source:
    prefix=source.read(min(4096,size));offset=binary.find(prefix);assert offset>=0,f'Embedded asset missing: {file}'
    source.seek(0);expected=hashlib.file_digest(source,'sha256').hexdigest()
   assert hashlib.sha256(memoryview(binary)[offset:offset+size]).hexdigest()==expected,f'Embedded bytes differ: {file}'
   checks.append({'asset':str(file.relative_to(root)).replace('\\','/'),'bytes':size,'sha256':expected})
  digest=hashlib.sha256(binary).hexdigest()
 handle=k.LoadLibraryExW(str(exe),None,2);assert handle
 try:
  resource=k.FindResourceW(handle,1,24);assert resource,'Windows manifest missing'
  xml=ctypes.string_at(k.LockResource(k.LoadResource(handle,resource)),k.SizeofResource(handle,resource)).decode('utf-8')
  level='requireAdministrator' if adapter=='vpn' else 'asInvoker';assert f'level="{level}"' in xml;assert 'permonitorv2' in xml.lower()
 finally:k.FreeLibrary(handle)
 results.append({'exe':exe.name,'architecture':'amd64','executionLevel':level,'perMonitorV2':True,'sha256':digest,'embeddedAssets':checks})
(review/'reports/binary-verification.json').write_text(json.dumps(results,indent=2),encoding='utf-8')
print(f'PASS: {len(results)} AMD64 executables; exact payload, font and ribbon bytes; UAC and DPI manifests')
