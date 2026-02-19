package metrics

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type systemCollector struct{}

func NewCollector() Collector {
	return &systemCollector{}
}

func (c *systemCollector) Collect(ctx context.Context) (Metrics, error) {
	var metrics Metrics

	cpu, err := c.collectCPU()
	if err != nil {
		return Metrics{}, err
	}
	metrics.CPU = cpu

	mem, err := c.collectMemory()
	if err != nil {
		return Metrics{}, err
	}
	metrics.Memory = mem

	disk, err := c.collectDisk()
	if err != nil {
		return Metrics{}, err
	}
	metrics.Disk = disk

	net, err := c.collectNetwork()
	if err != nil {
		return Metrics{}, err
	}
	metrics.Network = net

	metrics.Timestamp = time.Now()

	return metrics, nil
}

type cpuStat struct{ idle, total uint64 }

func readCPUStat() (cpuStat, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return cpuStat{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return cpuStat{}, scanner.Err()
	}

	line := scanner.Text()
	parts := strings.Fields(line)
	if len(parts) < 5 || parts[0] != "cpu" {
		return cpuStat{}, fmt.Errorf("unexpected /proc/stat format: %q", line)
	}

	// Sum all fields (user, nice, system, idle, iowait, irq, softirq, steal, …)
	// to get accurate total jiffies. Field index 4 is idle.
	var idle, total uint64
	for i, field := range parts[1:] {
		v, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuStat{}, fmt.Errorf("parse /proc/stat field %d: %w", i+1, err)
		}
		total += v
		if i == 3 { // index 3 (0-based within value fields) is idle
			idle = v
		}
	}
	return cpuStat{idle: idle, total: total}, nil
}

// collectCPU takes two snapshots 200ms apart to compute current CPU usage.
// A single snapshot gives cumulative averages since boot, not current load.
func (c *systemCollector) collectCPU() (float64, error) {
	s1, err := readCPUStat()
	if err != nil {
		return 0, err
	}

	time.Sleep(200 * time.Millisecond)

	s2, err := readCPUStat()
	if err != nil {
		return 0, err
	}

	deltaIdle := s2.idle - s1.idle
	deltaTotal := s2.total - s1.total
	if deltaTotal == 0 {
		return 0, nil
	}

	percent := float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
	return percent, nil
}

func (c *systemCollector) collectMemory() (MemoryMetrics, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemoryMetrics{}, err
	}
	defer file.Close()

	var memTotal, memAvailable uint64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}

		switch parts[0] {
		case "MemTotal:":
			memTotal = value * 1024
		case "MemAvailable:":
			memAvailable = value * 1024
		}
	}

	if memTotal == 0 {
		return MemoryMetrics{}, scanner.Err()
	}

	used := memTotal - memAvailable
	percent := float64(used) / float64(memTotal) * 100

	return MemoryMetrics{
		Used:      used,
		Total:     memTotal,
		Available: memAvailable,
		Percent:   percent,
	}, nil
}

func (c *systemCollector) collectDisk() (DiskMetrics, error) {
	var stat unix.Statfs_t
	wd, err := os.Getwd()
	if err != nil {
		return DiskMetrics{}, err
	}

	if err := unix.Statfs(wd, &stat); err != nil {
		return DiskMetrics{}, err
	}

	total := uint64(stat.Blocks) * uint64(stat.Bsize)
	free := uint64(stat.Bfree) * uint64(stat.Bsize)
	used := total - free
	percent := float64(used) / float64(total) * 100

	return DiskMetrics{
		Used:    used,
		Total:   total,
		Free:    free,
		Percent: percent,
	}, nil
}

func (c *systemCollector) collectNetwork() (NetworkMetrics, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return NetworkMetrics{}, err
	}
	defer file.Close()

	var totalSent, totalRecv, totalPktSent, totalPktRecv uint64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 10 {
			continue
		}

		recv, _ := strconv.ParseUint(parts[1], 10, 64)
		recvPackets, _ := strconv.ParseUint(parts[2], 10, 64)
		sent, _ := strconv.ParseUint(parts[9], 10, 64)
		sentPackets, _ := strconv.ParseUint(parts[10], 10, 64)

		totalRecv += recv
		totalPktRecv += recvPackets
		totalSent += sent
		totalPktSent += sentPackets
	}

	if err := scanner.Err(); err != nil {
		return NetworkMetrics{}, err
	}

	return NetworkMetrics{
		BytesSent:   totalSent,
		BytesRecv:   totalRecv,
		PacketsSent: totalPktSent,
		PacketsRecv: totalPktRecv,
	}, nil
}
