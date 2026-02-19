package metrics

import (
	"bufio"
	"context"
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

func (c *systemCollector) collectCPU() (float64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, scanner.Err()
	}

	line := scanner.Text()
	parts := strings.Fields(line)
	if len(parts) < 5 || parts[0] != "cpu" {
		return 0, nil
	}

	user, _ := strconv.ParseUint(parts[1], 10, 64)
	nice, _ := strconv.ParseUint(parts[2], 10, 64)
	system, _ := strconv.ParseUint(parts[3], 10, 64)
	idle, _ := strconv.ParseUint(parts[4], 10, 64)

	total := user + nice + system + idle
	if total == 0 {
		return 0, nil
	}

	percent := float64(total-idle) / float64(total) * 100
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
