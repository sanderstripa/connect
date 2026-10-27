# CONNECT graphite/glass icon specification

The approved first-column C and connection point replace the folded diamond.
App, Start menu, desktop, installer and extension: dark squircle, silver glyph.
Running Windows window/taskbar and system tray: silver while disconnected, turquoise only when core, tunnel and routing are ready. The tray has no plate.

Editable masters: `build/branding/source/*.svg`. Regenerate with `node scripts/render-app-icons.cjs` (development dependency: sharp), then `node scripts/package-icons.mjs` for macOS ICNS. Every PNG size is rendered directly from vector geometry. Windows keeps the existing shortcut refresh notifications.
