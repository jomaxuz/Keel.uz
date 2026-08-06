// Package sysstat reports what the machine has left.
//
// Read from /proc and statfs rather than from a monitoring agent: this is one
// VPS running one platform, and the question it answers is small and specific
// — "am I about to run out of memory or disk?" — which arrives long before
// anybody wants Prometheus.
//
// ⚠️ **These are the host's numbers, not the container's.** A normal Docker
// container shares the host's /proc, so meminfo and stat report the whole
// machine. That is what is wanted here (the control plane is asking about the
// server it and every tenant sit on) but it is worth knowing, because the same
// code inside a memory-limited container would report the host's memory and
// quietly mislead.
package sysstat

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Stats is one reading of the machine.
type Stats struct {
	// Busy CPU across all cores, 0–100. Measured over a short window rather
	// than since boot: a load average is a different question and an
	// average-since-boot is useless after a week of uptime.
	CPUPercent float64 `json:"cpuPercent"`
	Cores      int     `json:"cores"`
	// 1, 5 and 15 minute load averages. Kept beside the percentage because
	// they answer what it cannot: whether the machine is *queueing*.
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`

	// Bytes.
	MemTotal int64 `json:"memTotal"`
	// What is actually reclaimable — MemAvailable, not "free". Free memory on
	// a healthy Linux box is near zero because the page cache uses the rest,
	// and reporting it would make every server look like it is dying.
	MemAvailable int64   `json:"memAvailable"`
	MemPercent   float64 `json:"memPercent"`

	SwapTotal int64 `json:"swapTotal"`
	SwapUsed  int64 `json:"swapUsed"`

	DiskTotal   int64   `json:"diskTotal"`
	DiskFree    int64   `json:"diskFree"`
	DiskPercent float64 `json:"diskPercent"`
	// Which path the disk figures describe, so a wrong mount is visible
	// rather than silently reporting the container's own overlay.
	DiskPath string `json:"diskPath"`

	UptimeSeconds int64     `json:"uptimeSeconds"`
	At            time.Time `json:"at"`
	// Anything that could not be read. Reported rather than defaulted: a zero
	// that means "unknown" reads as "empty", and on a disk gauge those are
	// opposite emergencies.
	Errors []string `json:"errors,omitempty"`
}

// Read takes a measurement. `diskPath` is the filesystem to report on.
func Read(diskPath string) Stats {
	s := Stats{At: time.Now(), DiskPath: diskPath}

	if n, err := cpuCount(); err == nil {
		s.Cores = n
	}
	if pct, err := cpuPercent(200 * time.Millisecond); err == nil {
		s.CPUPercent = pct
	} else {
		s.Errors = append(s.Errors, "cpu: "+err.Error())
	}
	if a, b, c, err := loadAvg(); err == nil {
		s.Load1, s.Load5, s.Load15 = a, b, c
	}
	if err := readMem(&s); err != nil {
		s.Errors = append(s.Errors, "mem: "+err.Error())
	}
	if err := readDisk(&s, diskPath); err != nil {
		s.Errors = append(s.Errors, "disk: "+err.Error())
	}
	if up, err := uptime(); err == nil {
		s.UptimeSeconds = up
	}
	return s
}

// cpuPercent measures over a window. Two readings of /proc/stat and the
// difference between them — the only way to get a *current* figure, since the
// counters themselves are cumulative since boot.
func cpuPercent(window time.Duration) (float64, error) {
	idle1, total1, err := cpuTimes()
	if err != nil {
		return 0, err
	}
	time.Sleep(window)
	idle2, total2, err := cpuTimes()
	if err != nil {
		return 0, err
	}
	dt, di := total2-total1, idle2-idle1
	if dt <= 0 {
		return 0, nil
	}
	pct := float64(dt-di) / float64(dt) * 100
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct, nil
}

func cpuTimes() (idle, total int64, err error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 5 || fields[0] != "cpu" {
			continue
		}
		for i, v := range fields[1:] {
			n, e := strconv.ParseInt(v, 10, 64)
			if e != nil {
				continue
			}
			total += n
			// Fields 4 and 5 (idle, iowait) are both time the CPU was not
			// doing work anybody asked for.
			if i == 3 || i == 4 {
				idle += n
			}
		}
		return idle, total, nil
	}
	return 0, 0, os.ErrNotExist
}

func cpuCount() (int, error) {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "processor") {
			n++
		}
	}
	return n, nil
}

func loadAvg() (a, b, c float64, err error) {
	raw, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, err
	}
	f := strings.Fields(string(raw))
	if len(f) < 3 {
		return 0, 0, 0, os.ErrInvalid
	}
	a, _ = strconv.ParseFloat(f[0], 64)
	b, _ = strconv.ParseFloat(f[1], 64)
	c, _ = strconv.ParseFloat(f[2], 64)
	return a, b, c, nil
}

func readMem(s *Stats) error {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return err
	}
	defer f.Close()

	vals := map[string]int64{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) == 0 {
			continue
		}
		n, e := strconv.ParseInt(fields[0], 10, 64)
		if e != nil {
			continue
		}
		vals[parts[0]] = n * 1024 // meminfo is in kB
	}
	s.MemTotal = vals["MemTotal"]
	// MemAvailable, deliberately: "MemFree" on a healthy Linux box is near
	// zero because the page cache holds the rest, and a dashboard built on it
	// shows every server as critically low forever.
	s.MemAvailable = vals["MemAvailable"]
	if s.MemTotal > 0 {
		s.MemPercent = float64(s.MemTotal-s.MemAvailable) / float64(s.MemTotal) * 100
	}
	s.SwapTotal = vals["SwapTotal"]
	s.SwapUsed = vals["SwapTotal"] - vals["SwapFree"]
	return nil
}

func readDisk(s *Stats, path string) error {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return err
	}
	bs := int64(st.Bsize)
	s.DiskTotal = int64(st.Blocks) * bs
	// Bavail, not Bfree: the last few percent are reserved for root and a
	// service that is not root cannot use them. Counting them as free is how
	// a disk "with 3% left" stops accepting uploads.
	s.DiskFree = int64(st.Bavail) * bs
	if s.DiskTotal > 0 {
		s.DiskPercent = float64(s.DiskTotal-s.DiskFree) / float64(s.DiskTotal) * 100
	}
	return nil
}

func uptime() (int64, error) {
	raw, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	f := strings.Fields(string(raw))
	if len(f) == 0 {
		return 0, os.ErrInvalid
	}
	sec, _ := strconv.ParseFloat(f[0], 64)
	return int64(sec), nil
}
