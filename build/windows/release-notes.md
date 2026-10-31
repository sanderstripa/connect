Connect 0.3.2

- Update bundled Sudoku core and protocol parsing to upstream v0.5.2-1, including SOCKS5 UDP/IPv6 fixes and MPTCP mitigation.
- Preserve traffic counters, DIRECT exceptions, TUN bypass and TCP half-close on the new client architecture.
- Publish a separately verified compatible Sudoku component for in-app updates, using the protocol version in its package name.
- Prefer the bundled compatible core when a previous raw upstream update lacks desktop adapters; retain its files and saved credentials.
- Keep other protocols, profiles, subscription imports, extension integration and branding unchanged.
