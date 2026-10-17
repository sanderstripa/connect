# CONNECT UI System v1.0 implementation

Scope: CONNECT VPN desktop UI only. Screenshot PR #1, Screencast, Torrent, the Chrome popup and tray composition are unchanged.

Reference: https://github.com/sanderstripa/connect/blob/connect-ui-system-v1-reference/docs/CONNECT-UI-SYSTEM-v1.md

## Preserved

The original frontend/src/style.css, Montserrat.ttf and main.go remain unchanged. Their baseline hashes are regression-tested (line endings normalized for text). This preserves the existing palette, typography asset, 410×660 native dimensions, custom titlebar, minimize/hide-to-tray behavior and network lifecycle. The current effective titlebar height is 54 CSS px, with existing compact viewport overrides; the reference document's older 62px declaration is not imposed over the actual cascade.

| Token | Light | Dark |
| --- | --- | --- |
| Background | #f7f7f5 | #0c0c0d |
| Text | #171a1d | #ededee |
| Muted | #80858b | #81858c |
| Line | #dadbdc | #303236 |
| Surface | #eee | #1a1b1e |
| Accent | #171a1d | #e6e7e9 |
| Inverse | #fff | #121315 |
| Success | #438067 | #8cbaa2 |
| Error | #aa4242 | #e69595 |

## Shared controls

Metrics-only overrides are in frontend/src/styles/connect-ui-v1.css. Primary actions use the original accent and inverse tokens; secondary actions use the original neutral surface and line. Inputs and selects share 36px height, 8px radius and 13px content type. A narrow viewport keeps CONNECT's existing compact density. Switches use 40×23px monochrome tracks with circular thumbs. Focus rings are 2px; transitions are 140ms and respect reduced motion. Reusable slider/progress styling uses a 2px/12px slider and a separate 4px indeterminate progress track. No slider setting is added to the application.

The existing home screen and native header composition stay intact. Language preferences move from the titlebar into Appearance, as explicit RU/EN buttons beside the existing System/Light/Dark controls. The existing save/persistence methods and routing draft behavior are reused. Updates receive contrasting installation buttons, neutral rollback actions, a thin select arrow and real busy-state progress. Existing custom import and destructive-delete confirmation UI use the same control shapes. Removed update confirmations remain removed.

## Approved gear #8

The reference branch contains the specification but no original ten-icon sheet, raster or approved SVG for gear #8. No such asset was available in the project references. The existing CONNECT settings-entry icon is retained with no background; it is not represented as the approved gear and is not replaced with an invented alternative. Exact gear replacement remains pending its original approved visual.

## Verification limits

Windows Chromium tests exercise actual CSS rendering at 410×660 and compact effective viewport sizes, both original palettes, theme/language persistence, keyboard focus and routing, import, update/rollback and reduced motion. Native shell source and dimensions remain unchanged. Launching a second installed CONNECT UI would activate the user's existing singleton, so the production installation is not opened or altered during UI testing. Existing native helper/core tests run in isolated profiles. This is not a fresh system install/uninstall or a live VPN test.
