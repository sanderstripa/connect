Connect 0.1.6 · Windows 10/11 x64

### Connection screen

Removed the duplicate Sudoku / Xray / Mist buttons. Click the connection card to choose a saved profile.

### Settings → Updates

- Installed versions, individual checks and a Check all action for Connect, Xray, GeoIP/GeoSite, Sudoku and Mist.
- Stable and pre-release packages, including Xray pre-releases. Pre-release checking is enabled by default and can be turned off.
- Explicit version selection, SHA-256-verified downloads and immutable component slots. Component changes take effect on the next connection and also apply to the independent Chrome helper.
- Local rollback to the previous component version, preserved across application restarts. Geo can also roll back to its initial uninstalled state.
- Application updates and older versions launch the normal installer after Connect closes. Installed copies keep the previous installer for offline rollback. If no local backup is available, an older published installer can be selected after checking releases.

Mist updates use Connect-compatible packages with the SOCKS adapter; ordinary upstream Mist executables are not interchangeable. The separate Mist package is an updater asset, not a standalone application installer. GeoIP/GeoSite packages come from Loyalsoldier/v2ray-rules-dat. Packages without a published SHA-256 digest are not offered.

### Updating

Use Connect-Setup.exe for this update. Quit Connect from the tray first and disable the Connect extension on chrome://extensions while replacing the application. Reload/re-enable the extension after installation. Profiles are preserved. Core updates do not overwrite an executable used by a running connection.

### Validation

Windows tests cover verified installation, restart persistence, rollback, failed checksums, archive selection, tamper detection, core selection and a real installer-launch handoff using an isolated probe executable. Interface tests cover the simplified connection screen and the update menu in Russian and English. Real Xray, Sudoku, Geo and application release downloads were also checked in a temporary directory; Xray 26.10.10 was launched with its version command. No user connection or system installation was changed by these checks.

Mist remains experimental. This release does not claim to fix the previously reported intermittent connectivity issue. Executables are unsigned; no server credentials are bundled.
