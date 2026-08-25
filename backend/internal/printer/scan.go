package printer

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

// ---- Finding a printer nobody installed ----
//
// ⚠️ **The spooler only knows printers Windows was told about, and most
// restaurant printers are not.** An XP-Q80A or an Epson TM on the network takes
// raw ESC/POS on port 9100 — no driver, no Windows printer, nothing to
// enumerate. So `EnumPrinters` finds the four things somebody once installed
// and misses the one bolted under the pass, which is exactly the printer the
// person setting the till up is holding the box of.
//
// The address is on the printer's own self-test slip, and asking somebody to
// read it off a roll of paper works — right up until the slip is in the bin and
// the printer is behind a fridge. So the machine looks for itself.
//
// ⚠️ **What this proves is "something answers on 9100 here", not "this is a
// printer".** Nothing on that port identifies itself, and a device that takes a
// TCP connection may be anything. The screen therefore offers an address to
// try, and the test print is what confirms it — the same rule the rest of this
// package follows: paper is the only evidence.

// Found is one address that accepted a connection.
type Found struct {
	IP string `json:"ip"`
	// The port it answered on. Kept because the second one exists in the wild.
	Port int `json:"port"`
	// The interface's own address, so a machine on two networks can say which
	// side of it the printer is on.
	Via string `json:"via,omitempty"`
}

// ⚠️ **9100 first and 9101 after it.** 9100 is the raw-printing convention and
// almost every receipt printer ships on it; 9101 turns up on multi-port print
// servers and on a few models' second interface. Anything beyond those two is
// guessing, and every extra port doubles a scan somebody is standing in front
// of.
var scanPorts = []int{9100, 9101}

// Scan looks for printers on every IPv4 network this machine is on.
//
// ⚠️ **Only /24 and smaller.** A machine on a /16 would be 65 000 connections,
// which is a scan that never finishes and looks to a router like something it
// should block. Restaurant networks are /24; a larger one is left to the
// address field, which has been there all along.
func Scan(ctx context.Context) []Found {
	out := []Found{}
	nets := localNets()
	if len(nets) == 0 {
		return out
	}

	type job struct{ ip, via string }
	jobs := make(chan job)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// ⚠️ 64 at a time, and the number is a compromise rather than a maximum.
	// A monoblock with four gigabytes is doing this while somebody watches, and
	// opening 254 sockets at once is how a cheap router starts dropping the
	// till's own traffic.
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				for _, port := range scanPorts {
					if !answers(ctx, j.ip, port) {
						continue
					}
					mu.Lock()
					out = append(out, Found{IP: j.ip, Port: port, Via: j.via})
					mu.Unlock()
					// One printer, one row: a device listening on both ports is
					// still one printer, and listing it twice is a choice
					// somebody has to make for no reason.
					break
				}
			}
		}()
	}

	for _, n := range nets {
		for _, ip := range hosts(n) {
			select {
			case <-ctx.Done():
			case jobs <- job{ip: ip, via: n.IP.String()}:
			}
		}
	}
	close(jobs)
	wg.Wait()

	sort.Slice(out, func(i, j int) bool { return less(out[i].IP, out[j].IP) })
	return out
}

// answers reports whether something took a TCP connection there.
//
// ⚠️ **Opened and closed, nothing written.** Sending bytes to an unknown device
// to see what it is would mean printing on whatever it turns out to be — and a
// scan that spits blank paper out of every printer in the building is worse
// than no scan.
func answers(ctx context.Context, ip string, port int) bool {
	// ⚠️ 400ms: long enough for a switch on the same wire, short enough that
	// 254 addresses across 64 workers finish in a couple of seconds. A silent
	// host does not refuse — it says nothing — so this timeout *is* the scan's
	// running time.
	d := net.Dialer{Timeout: 400 * time.Millisecond}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// localNets returns this machine's own IPv4 networks, /24 or smaller.
func localNets() []*net.IPNet {
	var out []*net.IPNet
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		// ⚠️ Loopback skipped, and interfaces that are down: a disconnected
		// wifi adapter keeps its last address, and scanning it is 254 timeouts
		// spent proving nothing.
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			n, ok := a.(*net.IPNet)
			if !ok || n.IP.To4() == nil {
				continue
			}
			ones, bits := n.Mask.Size()
			if bits != 32 || ones < 24 {
				continue
			}
			out = append(out, &net.IPNet{IP: n.IP.To4(), Mask: n.Mask})
		}
	}
	return out
}

// hosts lists the addresses in a network, without the network and broadcast
// addresses and without this machine's own.
func hosts(n *net.IPNet) []string {
	var out []string
	ip := n.IP.Mask(n.Mask).To4()
	if ip == nil {
		return out
	}
	base := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
	ones, _ := n.Mask.Size()
	count := uint32(1) << (32 - ones)
	self := n.IP.To4()
	selfN := uint32(self[0])<<24 | uint32(self[1])<<16 | uint32(self[2])<<8 | uint32(self[3])
	for i := uint32(1); i < count-1; i++ {
		v := base + i
		if v == selfN {
			continue
		}
		out = append(out, fmt.Sprintf("%d.%d.%d.%d",
			byte(v>>24), byte(v>>16), byte(v>>8), byte(v)))
	}
	return out
}

func less(a, b string) bool {
	ai, bi := net.ParseIP(a).To4(), net.ParseIP(b).To4()
	if ai == nil || bi == nil {
		return a < b
	}
	for i := 0; i < 4; i++ {
		if ai[i] != bi[i] {
			return ai[i] < bi[i]
		}
	}
	return false
}
