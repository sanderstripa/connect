CONNECT FOR CHROME

1. Run Connect-Setup.exe and leave Chrome extension enabled.
2. Open chrome://extensions and enable Developer mode.
3. Choose Load unpacked and select C:\Program Files\Connect\ChromeExtension.
4. Pin Connect to the toolbar. Open Settings and import your server.
5. Use the single power button to connect or disconnect.

The desktop window does not need to be running. Chrome starts its own local
helper. Profiles are separate: %APPDATA%\connect-chrome\config.json.
Supported: Sudoku links/subscriptions, VLESS/VMess/Trojan including xhttp,
and MIST client JSON (or mist:// Base64-encoded client JSON).

This component protects Chrome proxy-compatible traffic, not the whole OS.
WebRTC non-proxied UDP is disabled while enabled. A failed helper leaves the
proxy set until you disconnect, preventing a silent return to direct browsing.

The ZIP is for manual loading after installing the native helper.
Chrome does not allow an ordinary Windows installer to silently install
an unpacked extension. Chrome Web Store publication is not part of this build.
