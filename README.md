<p align="center">
  <img src="build/appicon.png" width="96" alt="Connect icon">
</p>
<h1 align="center">Connect</h1>
<p align="center">A minimal VPN for Windows, with an independent Chrome extension.</p>
<p align="center"><a href="https://github.com/sanderstripa/connect/releases">Download for Windows</a></p>

Connect brings **Sudoku, Xray and experimental Mist** into a clean interface inspired by [sudoku-app](https://github.com/sanderstripa/sudoku-app). Montserrat typography, a quiet monochrome palette and light, dark or system themes keep the focus on your connection.

## Windows

- Connect with one button and switch between saved protocol profiles.
- Route device traffic through TUN; a green tray icon indicates an active core and tunnel.
- Import Sudoku links/subscriptions, VLESS, VMess, Trojan or a Mist client configuration.
- Preserve Xray xhttp parameters, including host, path, mode and extra, plus TLS/Reality settings. TCP, WebSocket and gRPC are also supported.
- Keep profiles across updates and uninstallations. No server credentials are bundled.

Install **Connect-Setup.exe**, add your own connection, then select its protocol and connect. Closing the window keeps Connect in the tray; choosing Exit disconnects it. The interface runs normally, while its network helper requests Windows elevation when needed.

Sudoku ↔ Xray switching retains TUN, DNS and routes when server addresses are IPv4 and the local port is shared. The new core is checked with real HTTP traffic; failure restores the previous profile. Domain names, IPv6, different ports and Mist use a full restart.

Requires Windows 10/11 x64 and Microsoft Edge WebView2 Runtime. Desktop profiles are stored in `%APPDATA%\connect-app`.

## Chrome

**Your connections, directly in the popup.** Click **Add**, paste a link or client configuration, and let Connect detect the protocol. Each saved connection has a small switch. Turning on another connection turns the previous one off; turning off the active switch disconnects. There is no separate settings page.

Chrome starts its own local helper, so the desktop window can remain closed. Browser profiles are stored independently in `%APPDATA%\connect-chrome`. The helper does not change Windows proxy settings, DNS or routes.

Enable the Chrome component in the installer. It opens `chrome://extensions` and copies the extension folder path. Enable **Developer mode**, choose **Load unpacked**, and select `C:\Program Files\Connect\ChromeExtension`. After updating, reload Connect once on that page. Existing profiles are preserved. Chrome Web Store distribution is not included.

The extension handles proxy-compatible Chrome traffic, rather than whole-device traffic. WebRTC UDP outside the proxy is disabled while enabled. If the helper or connection fails, the proxy remains in place to prevent a silent return to direct browsing; Connect retries, or you can explicitly disconnect. All protocols require the installed native helper.

## Validation and current limits

Automated checks cover Windows core and installer tests, link import, browser isolation, extension lifecycle and UI behavior. Local integration tests pass real HTTP through Xray/xhttp and Sudoku, exercise switching and rollback, and verify that Windows proxy settings remain unchanged. The Mist SOCKS adapter has TCP/UDP tests; an external Mist Edge connection and TUN operation in the user's network still need separate validation.

- Mist is experimental. Its sources are pinned to revision `d4e7dacf22f9c3f4c5d762883544c102da0c6b49` of `mist-hardening-32`.
- Windows split tunneling is available for Sudoku and Xray. Mist uses all-traffic mode; Chrome currently has no configurable exclusions.
- One Chrome profile can be active per Windows user. Protocol cores update with the application.
- Executables are unsigned. Local test timings do not guarantee remote connection speed.

## Build

Windows x64, Go 1.27.0+, Node.js 24+, Git and internet access are required.

```powershell
./scripts/build-windows.ps1
node ./scripts/test-extension.mjs
node ./scripts/test-native.mjs
cd frontend
npx playwright install chromium
npm test
cd ..
./scripts/build-installer.ps1 -SkipApp
```

Outputs: `build/bin/Connect.exe`, `Connect-Setup.exe` and `Connect-Chrome.zip`.

Xray 26.3.27 is downloaded from its official release with archive SHA-256 verification. Mist builds from included sources; Sudoku 0.5.1 and HEV 2.17.1 are packaged by the build scripts. See [BUILD-PROVENANCE.md](BUILD-PROVENANCE.md), [THIRD_PARTY.md](THIRD_PARTY.md) and [LICENSE](LICENSE) for provenance and licensing. The original project's GPL-3.0-or-later terms and notices are retained.
