# CONNECT UI System v1.0 — reference for four Windows apps

**State:** Approved design principles; implementation details pending real Windows visual testing.  
**Scope:** CONNECT VPN, Screenshot, Screencast, Torrent.  
**Purpose:** unify shared UI components WITHOUT redesigning applications.

## Non-negotiable rules

1. **Keep Montserrat**; do not replace fonts, redraw approved product icons, or introduce a new brand palette.
2. **Preserve the original light/dark shades** from the existing sources and screenshots. Never replace them with pure #FFFFFF/#000000. Demo values are not authority.
3. **Custom Windows window shell**. Upper left: app name; upper right: minimize and close ONLY. No maximize and no redundant app icon by the title. Use the existing CONNECT VPN titlebar as visual reference. Preserve current window sizes and drag/minimize/close/Alt+F4/DPI behavior.
4. Unify **the same kind of control** across apps while retaining each app's unique functional layout and density. Do not move or remove operational features just to make layouts symmetrical.
5. Settings icon: **approved icon #8 in the user-reviewed sheet of ten variants**; flat, monochrome, NO background tile. Do not replace with a generic gear or improvise SVG geometry until the original graphic has been accurately vectorized.
6. Theme and language, where implemented, belong **inside settings**. Uniform language choices RU/EN, theme System/Light/Dark. Do not expose UI options that are not actually supported yet.

## Shared components — approved principles

| Component | Specification |
|---|---|
| Primary button | Contrasting monochrome fill with inverted text; light theme dark fill, dark theme light fill |
| Secondary button | Neutral existing surface, 1px border |
| Tertiary | Minimal text/icon action |
| Buttons and inputs | 36px candidate height, 8px candidate radius; actual font family and existing palette unchanged |
| Input / select | 1px line, 13px content text, identical shape; thin arrow without background |
| Switch | ~40×23px, monochrome contrast with circular thumb; same ON/OFF states for each app |
| Slider | 2px track, 12px circular thumb, dark in light theme/light in dark theme; no shadow or outline |
| Separator | 1px subdued line, only when logically needed |
| Spacing | 4/8/12/16/24px scale; 24px between independent sections, 8–12px between related controls |
| Interaction | Subtle hover/press, 2px focus ring, dimmed disabled state; ~120–150ms transitions respecting reduced motion |
| Dialogs | Compact custom modal, reuse primary/secondary buttons; confirm destructive changes |
| Notice | Small icon + terse description, colors only for real success/failure statuses |
| Progress | 4px monochrome track, no slider thumb |

**Typography:** Montserrat in all apps. Proposed roles: title around 18px, section 15–16px, body 13px, label 12px, help/status 11–12px. These are design candidates; **do not blindly overwrite the existing visual reference**. CONNECT VPN currently uses a 62px titlebar and a 16px wordmark, so earlier 44–48px header proposal cannot be treated as an exact mandate. Validate visually on real-size windows.

## Actual source-derived palette (do not overwrite)

| Application | Light background | Dark background | Code |
|---|---|---|---|
| CONNECT VPN | `#F7F7F5` | `#0C0C0D` | `frontend/src/style.css` |
| Screenshot | `#F5F5F5` | `#0A0A0A` | `src/settings.cpp` |
| Screencast | `#F7F7F5` | `#0B0B0C` | `src/main.cpp` |
| Torrent | `#F5F5F4` | `#0C0C0C` | `src/visual.h` |

These are deliberate existing differences, **not permission to impose any one app's hex values on the others**. The precise shared canonical palette needs visual approval after source comparison.

Source details:
- CONNECT: CSS Montserrat @font-face; `--text:#171a1d / #ededee`, `--line:#dadbdc / #303236`. Titlebar `62px`.
- Screenshot: DirectWrite Montserrat (fallback Segoe UI), title 19, body 13, label 11.5; original `ink:#111111 / #F4F4F4`, `border:#D9DADC / #333639`.
- Screencast: Win32 Montserrat, title 23, body 13, helper 11; original RGB palette in `App::applyTheme()`.
- Torrent: embedded Montserrat via DirectWrite; existing colors and components in `src/visual.h`.

## Implementation order and guardrails

1. Read actual source and document design tokens for **all** four apps.
2. Begin with a small, isolated, reversible screenshot change in a **draft PR**; perform Windows build and screenshot review before merging.
3. Reconstruct approved settings gear #8 from its *original approved visual* and obtain approval of exact SVG at native size before using it anywhere.
4. Separate functional language/theme work from visual work. Screencast currently has a Russian-only main UI; Screenshot's installer supports RU/EN but its settings window is Russian. Do not fake language toggles. CONNECT and Screencast support System/Light/Dark, while Screenshot and Torrent require additional functional logic for System mode.
5. Roll out shared controls and custom headers incrementally, one app at a time; run actual Windows checks, light/dark screenshots, DPI and keyboard regression tests before merging.

**Explicit exclusions:** change in app functionality, VPN protocol logic, torrent engine, screen capture behavior, window dimensions, unapproved icons or colors, new UI frameworks.

**Initial draft implementation:** [Screenshot button hierarchy PR](https://github.com/sanderstripa/screenshot/pull/1).

This specification is the source of truth for *approved principles*, not an authorization to silently modify unresolved design details.
