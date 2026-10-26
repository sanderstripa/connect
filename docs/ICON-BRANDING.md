# CONNECT Windows icon specification

Approved brand concept: **A — olive/graphite folded diamond**.

The app icon and tray icons intentionally have different containers:

| Surface | Art | Resource |
| --- | --- | --- |
| Windows application, shortcut, Start menu, taskbar, Explorer | Olive/graphite mark on a pale circular plate, transparent outside the circle | `build/windows/icon.ico` / `build/appicon.png` |
| CONNECT installer and uninstaller | Same circular app icon | `installer-core/products/vpn.ico` and Wails `build/windows/icon.ico` |
| System tray, disconnected or starting | Graphite/grey folded mark **without any plate** | `build/tray/off-{size}.png` |
| System tray, connected | Graphite/olive folded mark **without any plate** | `build/tray/on-{size}.png` |

Masters are in `build/branding/source/` and must be used when rebuilding the
assets; the older `build/branding/connect.svg` is retained for legacy SVG / macOS
consumers and is **not** the source of the Windows bitmap resources.

`scripts/build-branding.ps1` renders all app and tray PNG sizes and updates
installer/frontend/extension copies. `scripts/build-icon.ps1` then packages the
Windows ICO, macOS ICNS and the shared CONNECT Installer's VPN icon. Run
`node scripts/verify-branding.mjs` to verify the packaged resource wiring.

The tray switches to green only when the app's existing connection readiness
criteria confirm a running core, tunnel, and successfully configured route.
Failure or disconnection restores the neutral mark; do not use a button click
as a successful-connect signal.

Do not add square badges, status dots, text, or alternate icon geometry without
separate design approval. Existing pinned Windows shortcuts can cache the old
exe icon until Explorer refreshes them or they are repinned.
