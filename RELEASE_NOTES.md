CONNECT 0.2.9 - Windows 10/11 x64

- Approved olive/graphite variant A icons across the client, installer, shortcuts and extension.
- Circular application/installer icon; fully transparent standalone OFF/ON tray marks.
- Complete multi-size ICO resources and Windows shell refresh after upgrades.
- Existing confirmed-tunnel readiness and network behavior preserved.
- Added transparency, compiled executable icon and connection-state regression checks.
- Installer window test now waits for actual WebView2 runtime readiness.

Validated: Windows builds, app/core tests, real local Xray/Sudoku traffic, interface tests and installer lifecycle.
Manual system-TUN/leak and Windows shell visual acceptance of the new build remains outstanding.

Close CONNECT and disable the Chrome extension before upgrading. Windows binaries remain unsigned.

---

Connect 0.2.8 · Windows 10/11 x64

- Validate real HTTPS traffic through SOCKS before applying desktop TUN routes. A successful SOCKS reply alone no longer counts as a usable connection.
- Check two independent HTTPS endpoints concurrently. Keep certificate verification enabled, disallow redirects and direct fallback, and reject truncated responses.
- Preserve the previous connection, TUN and DNS when switching to a profile whose upstream accepts SOCKS but fails the traffic check.
- Verify real HTTPS through the specific Windows TUN adapter after DNS/routes are configured; roll back failed setup.
- Retry transient core/adapter readiness briefly within one deadline, without repeatedly disconnecting/reconnecting.
- Keep the desktop connection indicator pending until startup validation completes.
- Distinguish a Cloudflare browser challenge from a transport failure in HTTP probe diagnostics.

This release prevents an unusable profile from replacing a working desktop connection. It does not change server configuration, repair incompatible server keys, or guarantee removal of Cloudflare challenges on a VPN exit IP. Profiles, UI, themes, typography and window size are preserved.

Close Connect and disable the Chrome extension before installation, then reload and re-enable the extension. Windows binaries remain unsigned.

---

# Connect 0.2.7

- Единый установщик CONNECT Installer на Go/Wails; общий исходник хранится в `sanderstripa/connect/installer-core`.
- Кнопка «Принять и установить» находится на том же уровне, что в VPN, в обеих темах и на обоих языках.
- Удаление корректно завершается и до первого запуска приложения, когда записи автозапуска ещё нет.
- Сохранены установочные адаптеры, настройки и особенности продукта из проверенных пользователем локальных сборок.
- Изменения интерфейса и функциональности самого приложения не вносились, кроме номера версии.

Пользователь подтвердил установку всех пяти предыдущих локальных сборок. Автоматические проверки текущей сборки указаны в GitHub Actions; повторная ручная установка каждого финального SHA не заявляется.
