Connect 0.1.8 · Windows 10/11 x64

### AmneziaWG

- Added an AmneziaWG connection type to Windows and the independent Chrome extension. Profiles appear after import and use the existing connection controls.
- Import a .conf file, paste its contents, or use a complete self-hosted vpn:// key. awg:// Base64 configuration links are also accepted. Service keys requiring Amnezia API enrollment need to be exported as a complete AmneziaWG .conf profile first.
- Official amneziawg-go v3.1.20260828, pinned to b5928efb6ca19f0153958460c3d141f04abc5c2e. Preserves AWG 1/2/3 obfuscation fields, including AWG 3 header protection. Supports SOCKS TCP/UDP and DNS through the AWG tunnel. No direct DNS fallback.
- Settings → Updates includes verified Connect-compatible AmneziaWG packages and rollback. The separate AmneziaWG ZIP is an updater asset, not the application installer.

Windows AmneziaWG currently requires All traffic mode with custom bypass rules disabled. One peer per profile. AllowedIPs are respected. Shell hooks and unknown configuration fields are rejected. Desktop and Chrome allocate independent local ports; keys remain in local profiles. No credentials are bundled.

### Validation

Real encrypted loopback peers exercised AWG 1, 2 and 3 settings, TCP, UDP and tunneled DNS. Tests cover import, share decoding, configuration validation, runtime generation, Windows/Chrome file selection and profile toggles. The release workflow also runs existing core, native helper, branding and UI checks. No live external AmneziaWG server or workstation TUN changes were used for validation.

Install Connect-Setup.exe with Connect closed and its Chrome extension disabled; re-enable/reload the extension afterward. Existing profiles are preserved. Executables are unsigned. Mist remains experimental; no claim is made about fixing previous intermittent connectivity reports.
