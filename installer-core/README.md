# CONNECT Installer Core

One Go/Wails 3 module provides the installer window, HTML/CSS/JavaScript, Montserrat, themes, localisation and product-specific Windows adapters for CONNECT VPN, Screenshot, Screencast, Torrent and Soulu.

This directory in `sanderstripa/connect` is the only maintained UI source. Other repositories use `installer-core.json` to pin a full CONNECT commit and invoke `scripts/build-connect-installer.ps1`. No installer UI is copied into those repositories. Legacy native installers are retained as historical/test sources; release builds use this module.

Application payloads and generated Windows resources are build inputs, not Git files. Build the application first, then run from PowerShell:

```powershell
./build.ps1 -Product screenshot -PayloadPath C:/build/Screenshot.exe -Version 0.7.2 -OutFile C:/build/Screenshot-Setup.exe
```

Products: `vpn`, `screenshot`, `screencast`, `torrent`, `soulu`. VPN additionally requires `-ExtensionPath`. Torrent requires its NSIS payload and `-ExpectedExecutable` pointing at the same build's Torrent.exe. Screencast takes its complete portable ZIP. Soulu takes a complete verified CEF package directory or ZIP; Chromium/CEF is not rebuilt by this tool.

Go 1.26.4 or newer and Windows x64 are required. Wails is pinned to the VPN reference version. `go.sum` pins dependencies. Build output includes a `.build.json` with product/source versions and SHA256 values. Do not publish a different payload under an existing release asset name.

`ui-review.cjs` tests both themes, both languages and the Screenshot wizard with mocked system operations. It asserts that the settings installation button has exactly the VPN's vertical position in all five products. Set `PLAYWRIGHT_MODULE` if using an existing Playwright installation and `INSTALLER_REVIEW_DIR` for screenshots. The gallery is UI evidence, not an installation test.

The owner manually installed all five review builds and confirmed they work on 19 October 2026. The following release changes preserve those installation adapters and align the settings button at the VPN baseline. Automated checks do not claim a fresh manual installation of each subsequent SHA.
