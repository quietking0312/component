package mmen

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// Resource holds a snapshot of system resource usage.
type Resource struct {
	Timestamp int64         `json:"timestamp"`
	CPU       CPUInfo       `json:"cpu"`
	Memory    MemoryInfo    `json:"memory"`
	Disks     []DiskInfo    `json:"disks"`
	DiskIOs   []DiskIOInfo  `json:"disk_ios"`
	Networks  []NetworkInfo `json:"networks"`
}

// CPUInfo represents CPU usage statistics.
type CPUInfo struct {
	UsagePercent  float64   `json:"usage_percent"`  // total usage %
	CoreCount     int32     `json:"core_count"`     // logical cores
	PhysicalCount int32     `json:"physical_count"` // physical cores
	PerCore       []float64 `json:"per_core"`       // per-core usage %
}

// MemoryInfo represents memory usage statistics.
type MemoryInfo struct {
	Total       uint64  `json:"total"`        // bytes
	Used        uint64  `json:"used"`         // bytes
	Free        uint64  `json:"free"`         // bytes
	UsedPercent float64 `json:"used_percent"` // %
}

// DiskInfo represents disk partition usage.
type DiskInfo struct {
	Path        string  `json:"path"`
	Total       uint64  `json:"total"`        // bytes
	Used        uint64  `json:"used"`         // bytes
	Free        uint64  `json:"free"`         // bytes
	UsedPercent float64 `json:"used_percent"` // %
	FSType      string  `json:"fs_type"`
}

// DiskIOInfo represents disk I/O statistics.
type DiskIOInfo struct {
	Name       string  `json:"name"`
	ReadBytes  uint64  `json:"read_bytes"`
	WriteBytes uint64  `json:"write_bytes"`
	ReadCount  uint64  `json:"read_count"`
	WriteCount uint64  `json:"write_count"`
	ReadSpeed  float64 `json:"read_speed"`  // bytes/s
	WriteSpeed float64 `json:"write_speed"` // bytes/s
}

// NetworkInfo represents network interface statistics.
type NetworkInfo struct {
	Name        string  `json:"name"`
	BytesSent   uint64  `json:"bytes_sent"`
	BytesRecv   uint64  `json:"bytes_recv"`
	PacketsSent uint64  `json:"packets_sent"`
	PacketsRecv uint64  `json:"packets_recv"`
	SentSpeed   float64 `json:"sent_speed"` // bytes/s
	RecvSpeed   float64 `json:"recv_speed"` // bytes/s
}

// GetResource returns a snapshot of current resource usage (without I/O / network speed).
func GetResource() (*Resource, error) {
	return getResourceSnapshot()
}

// GetResourceWithSpeed samples resources twice (separated by interval) and calculates
// disk I/O and network transfer speeds.
func GetResourceWithSpeed(interval time.Duration) (*Resource, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("interval must be positive")
	}

	before, err := getResourceSnapshot()
	if err != nil {
		return nil, err
	}

	// prime CPU cache then wait
	_, _ = cpu.Percent(0, false)
	time.Sleep(interval)

	after, err := getResourceSnapshot()
	if err != nil {
		return nil, err
	}

	after.Timestamp = time.Now().Unix()
	calcSpeeds(before, after, interval)
	return after, nil
}

func getResourceSnapshot() (*Resource, error) {
	res := &Resource{Timestamp: time.Now().Unix()}

	cpuInfo, err := getCPUInfo()
	if err != nil {
		return nil, fmt.Errorf("cpu: %w", err)
	}
	res.CPU = *cpuInfo

	memInfo, err := getMemoryInfo()
	if err != nil {
		return nil, fmt.Errorf("memory: %w", err)
	}
	res.Memory = *memInfo

	disks, err := getDiskInfo()
	if err != nil {
		return nil, fmt.Errorf("disk: %w", err)
	}
	res.Disks = disks

	diskIOs, err := getDiskIOInfo()
	if err != nil {
		return nil, fmt.Errorf("disk io: %w", err)
	}
	res.DiskIOs = diskIOs

	nets, err := getNetworkInfo()
	if err != nil {
		return nil, fmt.Errorf("network: %w", err)
	}
	res.Networks = nets

	return res, nil
}

func getCPUInfo() (*CPUInfo, error) {
	percentages, err := cpu.Percent(0, true)
	if err != nil {
		return nil, err
	}

	infos, err := cpu.Info()
	if err != nil {
		return nil, err
	}

	var totalPercent float64
	for _, p := range percentages {
		totalPercent += p
	}
	if len(percentages) > 0 {
		totalPercent /= float64(len(percentages))
	}

	physicalCount := int32(0)
	seen := make(map[string]struct{})
	for _, info := range infos {
		if info.PhysicalID != "" {
			if _, ok := seen[info.PhysicalID]; !ok {
				seen[info.PhysicalID] = struct{}{}
				physicalCount++
			}
		}
	}
	if physicalCount == 0 {
		physicalCount = int32(len(infos))
	}

	return &CPUInfo{
		UsagePercent:  totalPercent,
		CoreCount:     int32(len(percentages)),
		PhysicalCount: physicalCount,
		PerCore:       percentages,
	}, nil
}

func getMemoryInfo() (*MemoryInfo, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	return &MemoryInfo{
		Total:       vm.Total,
		Used:        vm.Used,
		Free:        vm.Free,
		UsedPercent: vm.UsedPercent,
	}, nil
}

func getDiskInfo() ([]DiskInfo, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var result []DiskInfo
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue // skip inaccessible mounts
		}
		result = append(result, DiskInfo{
			Path:        p.Mountpoint,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
			FSType:      p.Fstype,
		})
	}
	return result, nil
}

func getDiskIOInfo() ([]DiskIOInfo, error) {
	counters, err := disk.IOCounters()
	if err != nil {
		return nil, err
	}

	var result []DiskIOInfo
	for name, c := range counters {
		result = append(result, DiskIOInfo{
			Name:       name,
			ReadBytes:  c.ReadBytes,
			WriteBytes: c.WriteBytes,
			ReadCount:  c.ReadCount,
			WriteCount: c.WriteCount,
		})
	}
	return result, nil
}

func getNetworkInfo() ([]NetworkInfo, error) {
	counters, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	var result []NetworkInfo
	for _, c := range counters {
		result = append(result, NetworkInfo{
			Name:        c.Name,
			BytesSent:   c.BytesSent,
			BytesRecv:   c.BytesRecv,
			PacketsSent: c.PacketsSent,
			PacketsRecv: c.PacketsRecv,
		})
	}
	return result, nil
}

func calcSpeeds(before, after *Resource, interval time.Duration) {
	secs := interval.Seconds()
	if secs <= 0 {
		return
	}

	beforeDiskMap := make(map[string]DiskIOInfo, len(before.DiskIOs))
	for _, d := range before.DiskIOs {
		beforeDiskMap[d.Name] = d
	}
	for i := range after.DiskIOs {
		if b, ok := beforeDiskMap[after.DiskIOs[i].Name]; ok {
			after.DiskIOs[i].ReadSpeed = float64(after.DiskIOs[i].ReadBytes-b.ReadBytes) / secs
			after.DiskIOs[i].WriteSpeed = float64(after.DiskIOs[i].WriteBytes-b.WriteBytes) / secs
		}
	}

	beforeNetMap := make(map[string]NetworkInfo, len(before.Networks))
	for _, n := range before.Networks {
		beforeNetMap[n.Name] = n
	}
	for i := range after.Networks {
		if b, ok := beforeNetMap[after.Networks[i].Name]; ok {
			after.Networks[i].SentSpeed = float64(after.Networks[i].BytesSent-b.BytesSent) / secs
			after.Networks[i].RecvSpeed = float64(after.Networks[i].BytesRecv-b.BytesRecv) / secs
		}
	}
}
