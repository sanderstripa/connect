# Build provenance

- Visual/application foundation: sanderstripa/sudoku-app, commit fcf7f68b4d31c17ecfe51605ee9b77ad3d4e6c0c.
- Sudoku core: SUDOKU-ASCII/sudoku v0.5.1, commit 4889b53cb35355123bebd40e6c76a9582de7c23d, with inherited source patches.
- HEV: 2.17.1 Windows x64 distribution, including Wintun and MSYS2 runtime.
- Xray: official v26.3.27 Windows x64 archive. SHA256: d004c39288ce9ada487c6f398c7c545f7d749e44bdfdd59dbc9f865afba4e1ad.
  Executable SHA256: 15c2d007954ac53ba69b80ec91242786b3c0b71d52649165b4ca1d5cc96ef8f1.
  This matches the executable in the user's working VlessXhttpChrome installation.
- Mist: sanderstripa/mist, mist-hardening-32, commit d4e7dacf22f9c3f4c5d762883544c102da0c6b49. Snapshot in engines/mist.
  Connect adds loopback SOCKS5 TCP/UDP using WireGuard's gVisor netstack and binds Windows transport sockets to the physical interface when TUN is active.
- Go/Wails/JavaScript dependencies are pinned in go.sum, engines/mist/go.sum, frontend/package-lock.json.
- Browser extension source is new. The user's existing extension was inspected for native messaging, proxy management, watchdog and WebRTC behavior. Credentials and profiles were not imported or included.
- Icon source: user-provided VPN.ico.

Release assets are unsigned. SHA256 checksums accompany the release.
