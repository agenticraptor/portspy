package ports

import (
	"syscall"
	"time"

	gnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// enumerate returns the raw listening sockets for the requested protocol filter
// ("", "tcp", or "udp"), using gopsutil for cross-platform socket discovery.
func enumerate(protoFilter string) ([]rawListener, error) {
	conns, err := gnet.Connections("inet")
	if err != nil {
		return nil, err
	}
	want, _ := normalizeProto(protoFilter)

	out := make([]rawListener, 0, len(conns))
	for _, c := range conns {
		var proto Proto
		switch c.Type {
		case syscall.SOCK_STREAM:
			if c.Status != "LISTEN" {
				continue
			}
			proto = ProtoTCP
		case syscall.SOCK_DGRAM:
			// A UDP socket with no connected remote peer is effectively a
			// listener (bound and ready to receive).
			if c.Raddr.Port != 0 {
				continue
			}
			proto = ProtoUDP
		default:
			continue
		}
		if want != "" && string(proto) != want {
			continue
		}
		out = append(out, rawListener{
			Proto: proto,
			Addr:  c.Laddr.IP,
			Port:  int(c.Laddr.Port),
			PID:   int(c.Pid),
		})
	}
	return out, nil
}

// procCache is a gopsutil-backed procSource that resolves each PID at most once.
type procCache struct {
	hits   map[int]procInfo
	misses map[int]bool
}

func newProcCache() *procCache {
	return &procCache{hits: map[int]procInfo{}, misses: map[int]bool{}}
}

// Lookup implements procSource.
func (p *procCache) Lookup(pid int) (procInfo, bool) {
	if pid <= 0 {
		return procInfo{}, false
	}
	if pi, ok := p.hits[pid]; ok {
		return pi, true
	}
	if p.misses[pid] {
		return procInfo{}, false
	}
	pi, ok := lookupProc(pid)
	if ok {
		p.hits[pid] = pi
	} else {
		p.misses[pid] = true
	}
	return pi, ok
}

// lookupProc reads process details via gopsutil. Each field is best-effort:
// some (cwd, exe) may be unavailable for processes owned by other users without
// elevated privileges, in which case they are simply left blank.
func lookupProc(pid int) (procInfo, bool) {
	pr, err := process.NewProcess(int32(pid))
	if err != nil {
		return procInfo{}, false
	}
	pi := procInfo{PID: pid}
	if name, err := pr.Name(); err == nil {
		pi.Name = name
	}
	if cmd, err := pr.Cmdline(); err == nil {
		pi.Command = cmd
	}
	if exe, err := pr.Exe(); err == nil {
		pi.Exe = exe
	}
	if cwd, err := pr.Cwd(); err == nil {
		pi.Cwd = cwd
	}
	if ts, err := pr.CreateTime(); err == nil && ts > 0 {
		pi.CreateTime = time.UnixMilli(ts)
	}
	pi.Parents = lineage(pr)
	return pi, pi.Name != "" || pi.Command != ""
}

// lineage walks up the parent chain, newest ancestor first, stopping at init
// (pid <= 1) or after a sane depth cap.
func lineage(pr *process.Process) []ProcRef {
	const maxDepth = 8
	var chain []ProcRef
	cur := pr
	for i := 0; i < maxDepth; i++ {
		parent, err := cur.Parent()
		if err != nil || parent == nil {
			break
		}
		ppid := int(parent.Pid)
		if ppid <= 1 {
			break
		}
		name, _ := parent.Name()
		chain = append(chain, ProcRef{PID: ppid, Name: name})
		cur = parent
	}
	return chain
}
