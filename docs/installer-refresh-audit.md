# CONNECT installer refresh: audit and verification

Baseline: `b5891ae68ead863e9d4fb04290492165cbff676c` (main, 2026-10-18).
Backup: `backup/installer-before-redesign-20261018`. Work: `feat/installer-visual-refresh`.
Scope: installer, installer build and installer-specific tests only. Application and extension sources are unchanged.

| Behavior | Baseline | Updated |
| --- | --- | --- |
| Payload | Embedded Connect.exe | Same embedding; local review build embeds SHA-verified public v0.2.2 |
| Destination | ProgramFiles/Connect only | Registered InstallLocation first; custom local folder for new installs; existing updates remain in place |
| Running app/helper | Atomic replacement reports a locked-file error | Windows lock preflight before mutations, same atomic replacement; exit through tray/disconnect extension then Retry; never terminate user processes |
| App-driven update | StopProxy, installer waits for UI exit, tray:quit | Unchanged application handoff |
| Desktop shortcut | Optional, public desktop | Same location and target, real switch |
| Start menu shortcut | Always created | Default on; optional; existing shortcuts are preserved when off |
| Chrome | Default on; unpack, native-host manifest and HKLM registration; copy path and open Chrome | Same files, origin and registration; off skips preparation, registration and browser/clipboard actions |
| Chrome manual work | Developer mode / Load unpacked | Preserved, plus explicit manual Update guidance; browser preparation failure retries without reinstalling |
| Uninstall | Remove EXE, native-host registration, shortcuts, autostart; keep profiles and extension folder | Same operations, resolved custom destination; only remove uninstaller/empty app folder after success |
| User data | AppData and remaining installation files retained | Unchanged; no recursive directory deletion |
| UI | 440x600; themes/languages; animated progress | 840x540; four screens; immediate RU/EN and neutral light/dark themes; factual completed-stage progress |
| Completion | Open or close | Done, with an actual launch switch; Explorer keeps normal launch integrity |
| Version | Hardcoded in HTML and Go | HTML receives Go version; build derives Go version from application package.json |

The user supplied vpn.png and explicitly authorized redrawing the ribbon from scratch. The bundled transparent PNG was generated from that visual reference. The user then explicitly removed the top-left icon; it is absent. Montserrat is the existing bundled font. Buttons/background/text use neutral colors; sage is confined to the ribbon.

## Local verification

- Original Windows Go tests run against the untouched backup worktree: 4 passed.
- Updated Windows tests: original authentication, atomic replace/data preservation, and embedded-PE checks, plus isolated install/update/remove, registry restoration, shortcuts, extension on/off, locked EXE refusal, concurrent progress reads, guarded post-install endpoints, and an actual inert running EXE that the installer must not terminate.
- Lifecycle tests use real Windows file operations, WScript shortcut creation, and a disposable HKCU registry subtree in place of HKLM. ProgramFiles/ProgramData/Public are temporary folders. Production profiles, native-host keys, autostart and network settings are not touched. Autostart cleanup is injected as a no-op in this local isolated test; the production cleanup command is retained unchanged.
- Browser integration: six tests cover both themes, RU/EN, editable/native-picker-result path, settings persistence between screens, all options off, default options, failures/retry, factual progress, disabled close during work, Chrome-only retry without repeating install, launch on/off, and uninstall. Browser tests stub native API responses; they do not prove Chrome profile integration.
- Windows production build uses existing requireAdministrator manifest and the normal build-installer.ps1 script.
- Review build payload: public Connect.exe v0.2.2, SHA-256 `0cbc6dc9d1baf34e762487312a3bb902a2667697bb33b0315663c013ef4b429e`, matched to the GitHub release asset digest. No application rebuild or application-source changes are included in the local review binary.

## Remaining release gates

The installed working CONNECT/Chrome on this computer must remain untouched. Therefore the local checks do not claim a complete interactive elevated install/uninstall in the user's real profile, UAC acceptance/cancellation, actual clipboard and Chrome Developer Mode loading/updating, or VPN-active shutdown with real user profiles. Native folder-dialog appearance, Windows minimize/Alt-F4 behavior, multiple-monitor DPI and an interactive clean-machine install/update/uninstall require manual review or an isolated Windows desktop. Screenshots are rendered from the exact installer HTML/CSS/JS in Chrome at 840x540, rather than captures of the elevated WebView2 window.

No tag, release, main merge or auto-merge is authorized. Keep the PR draft until Windows CI and the remaining interactive checks are reviewed and the user explicitly permits release.

Native preview attempt: a separate non-elevated copy was launched with a disposable WebView2 profile. It did not expose a usable window/CDP endpoint to the test harness, so native window rendering/minimize/folder-picker checks remain unverified. Only this test-owned preview process was stopped; installed CONNECT processes were left running. Do not infer a successful native launch from the successful build or Chrome-rendered screenshots.

## Release authorization: 2026-10-19

The user approved matte ribbon variant B and explicitly authorized publishing v0.2.3. This supersedes the earlier no-merge/no-release instruction after final checks pass. Release metadata is updated consistently; no VPN/application behavior changes are introduced. A native installer-window create/close smoke check is now required by the Windows workflow. The disposable review workflow using an already published payload was removed: release validation builds both application and installer from the same commit.

## Window dragging correction: v0.2.4

The globally declared async function chrome() in setup.js replaced window.chrome before the deferred Wails runtime loaded. WebView2 exposes its native bridge through window.chrome.webview, so dragging/native window controls could not invoke Windows. Rename the helper to prepareChrome(), retaining all API payloads and extension behavior. Free non-interactive window surfaces inherit the drag marker; buttons, inputs, labels and window controls explicitly opt out. The regression uses the actual bundled Wails runtime, a recording native bridge and real mouse gestures; it fails on the old global function and passes after the fix in both install and uninstall modes.

## Folder dialog cancellation correction: v0.2.5

Wails alpha.74 returns its internal cfd.ErrorCancelled ("cancelled by user") when a user closes the folder dialog without selecting a folder. The installer previously returned HTTP 500 for every non-nil dialog error, so the UI displayed a false failure warning. Normalize cancellation at this native-dialog boundary into a successful response with an empty path and cancelled=true. Handle the pinned Wails sentinel (including wrapped errors), Windows ERROR_CANCELLED, and COM HRESULT_FROM_WIN32(ERROR_CANCELLED); retain all other errors. Empty successful selections are also cancellation. The existing frontend preserves its current path when the response has no selected path and clears the old warning normally.

Windows API tests cover selection, empty cancellation, Wails/Windows/COM cancellation and real access/initialization failures; installer state must remain unchanged. Browser regressions in RU/EN cover cancelling with the default path, cancelling after manually editing the path, an actual-error response followed by cancellation retry, successful selection followed by cancellation, and installation using the retained selected path.

## Installer language and theme handoff (0.2.6)
Setup sends the currently selected language and theme with both install actions. After successful file/shortcut/registry operations, it atomically writes installer-preferences.json beside Connect.exe with a fresh random installation ID. The desktop backend consumes this after legacy profile migration and configuration load, in the actual desktop user's process. It changes only UI language/theme, saves through the existing atomic Store.Save, then records a per-user receipt. The same installation is applied once; every reinstall gets a new ID. Browser/native-host configuration is excluded. No elevated user's AppData is edited by setup. Missing preference files preserve portable/older installer behavior. Invalid preferences fail explicitly without replacing user configuration.
Validation: isolated native install/update/remove tests inspect the handoff file; store tests verify fresh/update overrides, unrelated configuration preservation, user edits on subsequent launches, and reinstallation. Browser tests assert selected RU/EN and light/dark values in installation requests.
