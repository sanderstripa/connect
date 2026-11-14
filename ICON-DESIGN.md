# Approved circular Connect icon

The approved 14 November 2026 preview is preserved in `build/branding/source/approved-round.png`. Transparent exterior, dark circular plate, silver arc and connection dot.

`scripts/render-app-icons.cjs` exports every application, installer and extension size from that master. SVG copies embed the same artwork. The connected-state vector uses a circular plate and preserves its existing green palette; tray glyphs stay transparent and retain their existing OFF/ON behavior. Run the renderer with the development-only `sharp` package, then `node scripts/package-icons.mjs` to package Windows ICO and macOS ICNS.

All installer product icons and versioned Windows shortcut icon files consume the same exports.
