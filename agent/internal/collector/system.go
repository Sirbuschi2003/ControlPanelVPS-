package collector

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type Metrics struct {
	Timestamp  time.Time     `json:"timestamp"`
	CPU        CPUMetrics    `json:"cpu"`
	Memory     MemoryMetrics `json:"memory"`
	Disk       DiskMetrics   `json:"disk"`
	Network    NetMetrics    `json:"network"`
	Uptime     uint64        `json:"uptime"`
	LoadAvg    LoadMetrics   `json:"load_avg"`
	Hostname   string        `json:"hostname"`
	OS         string        `json:"os"`
	KernelVer  string        `json:"kernel_version"`
}

type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
	Cores        int     `json:"cores"`
}

type MemoryMetrics struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	FreeBytes      uint64  `json:"free_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
}

type DiskMetrics struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type NetMetrics struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
}

type LoadMetrics struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

func Collect() (*Metrics, error) {
	m := &Metrics{Timestamp: time.Now()}

	cpuPct, err := cpu.Percent(500*time.Millisecond, false)
	if err == nil && len(cpuPct) > 0 {
		m.CPU.UsagePercent = cpuPct[0]
	}
	if count, err := cpu.Counts(true); err == nil {
		m.CPU.Cores = count
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		m.Memory.TotalBytes = vm.Total
		m.Memory.UsedBytes = vm.Used
		m.Memory.FreeBytes = vm.Free
		m.Memory.UsagePercent = vm.UsedPercent
	}

	if usage, err := disk.Usage("/"); err == nil {
		m.Disk.TotalBytes = usage.Total
		m.Disk.UsedBytes = usage.Used
		m.Disk.FreeBytes = usage.Free
		m.Disk.UsagePercent = usage.UsedPercent
	}

	if counters, err := net.IOCounters(false); err == nil && len(counters) > 0 {
		m.Network.BytesSent = counters[0].BytesSent
		m.Network.BytesRecv = counters[0].BytesRecv
	}

	if info, err := host.Info(); err == nil {
		m.Uptime = info.Uptime
		m.Hostname = info.Hostname
		m.OS = info.OS
		m.KernelVer = info.KernelVersion
	}

	if avg, err := load.Avg(); err == nil {
		m.LoadAvg.Load1 = avg.Load1
		m.LoadAvg.Load5 = avg.Load5
		m.LoadAvg.Load15 = avg.Load15
	}

	return m, nil
}
