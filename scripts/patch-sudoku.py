#!/usr/bin/env python3
"""Apply desktop adapters to pinned upstream v0.5.2-1 without replacing upstream handlers."""
import pathlib, shutil, sys
root = pathlib.Path(__file__).resolve().parent.parent
core = pathlib.Path(sys.argv[1])
for src in (root / 'scripts/sudoku_patches').rglob('*'):
    if src.is_file():
        dst = core / src.relative_to(root / 'scripts/sudoku_patches')
        dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(src, dst)
def replace(file, old, new):
    p = core / file
    s = p.read_text()
    if s.count(old) != 1:
        raise RuntimeError(f'{file}: expected exactly one upstream anchor: {old}')
    p.write_text(s.replace(old, new, 1))
replace('internal/app/client_target.go', 'return conn, decision, true', 'return wrapConnForTrafficStats(conn, decision.shouldProxy()), decision, true')
replace('internal/app/client_route.go', 'if d, ok := mgrs.match(routeActionDirect, destAddr, destIP); ok {', '''if ok, match := matchDirectAddress(mgrs, destAddr, destIP); ok {
            return routeDecision{action: routeActionDirect, match: match}
        }
        if d, ok := mgrs.match(routeActionDirect, destAddr, destIP); ok {''')
for args, mapped in [('s.reply, *clientAddr', 's.reply, net.UDPAddrFromAddrPort(*clientAddr), false'), ('req.payload, direct', 'req.payload, net.UDPAddrFromAddrPort(direct), true'), ('reply, *clientAddr', 'reply, net.UDPAddrFromAddrPort(*clientAddr), false')]:
    replace('internal/app/client_udp_session.go', 's.udpConn.WriteToUDPAddrPort(' + args + ')', 'udpWriteTo(s.udpConn, ' + mapped + ')')
if '//go:debug multipathtcp=0' not in (core / 'cmd/sudoku-tunnel/main.go').read_text():
    raise RuntimeError('Upstream MPTCP mitigation is missing')
