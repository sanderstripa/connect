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
