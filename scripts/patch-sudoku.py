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

# sync.Pool keeps a victim generation after one GC. Clear both generations so
# the idle-connection budget measures retained connections rather than pooled
# buffers left over by this package's other tests. Keep the 8 KiB assertion.
replace('pkg/crypto/idle_memory_test.go',
        '\truntime.GC()\n\truntime.ReadMemStats(&after)',
        '\truntime.GC()\n\truntime.GC()\n\truntime.ReadMemStats(&after)')
# The peer can finish reading before the writer returns its pooled frame.
# Wait for every write to finish before calling the connections idle.
replace('pkg/crypto/idle_memory_test.go', '"runtime"', '"runtime"\n\t"sync"')
replace('pkg/crypto/idle_memory_test.go', '\tvar conns []*RecordConn', '\tvar writers sync.WaitGroup\n\tvar conns []*RecordConn')
replace('pkg/crypto/idle_memory_test.go',
        '\t\t\tgo func(w *RecordConn) { _, _ = w.Write(payload) }(pair[0])',
        '\t\t\twriters.Add(1)\n\t\t\tgo func(w *RecordConn) { defer writers.Done(); _, _ = w.Write(payload) }(pair[0])')
replace('pkg/crypto/idle_memory_test.go',
        '\t}\n\truntime.GC()\n\truntime.GC()\n\truntime.ReadMemStats(&after)',
        '\t}\n\twriters.Wait()\n\truntime.GC()\n\truntime.GC()\n\truntime.ReadMemStats(&after)')
replace('pkg/crypto/idle_memory_test.go', '\tclose(stop)', '\tclose(stop)\n\tdefer func() { for _, c := range conns { _ = c.Close() } }()')
