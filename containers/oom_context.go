package containers

import (
	"sync"
	"time"

	"github.com/coroot/coroot-node-agent/node"
)

// OOMContext holds contextual information about OOM events
type OOMContext struct {
	Timestamp         time.Time `json:"timestamp"`
	MemoryPressure    string    `json:"memory_pressure"`     // none, low, medium, high, critical
	NodeMemoryUsage   float64   `json:"node_memory_usage"`   // percentage
	ContainerMemLimit uint64    `json:"container_mem_limit"` // bytes
	ContainerMemUsage uint64    `json:"container_mem_usage"` // bytes
	ProcessName       string    `json:"process_name"`
	ContainerName     string    `json:"container_name"`
	OOMScore          int       `json:"oom_score"`
}

// OOMContextCollector manages OOM context collection
type OOMContextCollector struct {
	recentOOMs map[uint32]*OOMContext // PID -> OOM context
	mutex      sync.RWMutex
	procRoot   string
}

// NewOOMContextCollector creates a new OOM context collector
func NewOOMContextCollector(procRoot string) *OOMContextCollector {
	return &OOMContextCollector{
		recentOOMs: make(map[uint32]*OOMContext),
		procRoot:   procRoot,
	}
}

// RecordOOM records an OOM event with context
func (occ *OOMContextCollector) RecordOOM(pid uint32, containerName, processName string, containerMemLimit, containerMemUsage uint64) *OOMContext {
	occ.mutex.Lock()
	defer occ.mutex.Unlock()

	context := &OOMContext{
		Timestamp:         time.Now(),
		ContainerName:     containerName,
		ProcessName:       processName,
		ContainerMemLimit: containerMemLimit,
		ContainerMemUsage: containerMemUsage,
	}

	// Get system pressure information
	if pressure, err := node.GetSystemPressure(occ.procRoot); err == nil {
		context.MemoryPressure = pressure.GetMemoryPressureLevel()
	} else {
		context.MemoryPressure = "unknown"
	}

	// Get node memory usage
	if memInfo, err := node.MemoryInfo(occ.procRoot); err == nil {
		if memInfo.TotalBytes > 0 {
			usedBytes := memInfo.TotalBytes - memInfo.AvailableBytes
			context.NodeMemoryUsage = (usedBytes / memInfo.TotalBytes) * 100
		}
	}

	// Get OOM score for the process (if still available)
	context.OOMScore = getOOMScore(pid)

	occ.recentOOMs[pid] = context

	// Clean up old entries (keep last 100 OOMs)
	if len(occ.recentOOMs) > 100 {
		oldestTime := time.Now()
		var oldestPid uint32
		for pid, ctx := range occ.recentOOMs {
			if ctx.Timestamp.Before(oldestTime) {
				oldestTime = ctx.Timestamp
				oldestPid = pid
			}
		}
		delete(occ.recentOOMs, oldestPid)
	}

	return context
}

// GetOOMContext retrieves OOM context for a PID
func (occ *OOMContextCollector) GetOOMContext(pid uint32) *OOMContext {
	occ.mutex.RLock()
	defer occ.mutex.RUnlock()
	return occ.recentOOMs[pid]
}

// GetRecentOOMs returns all recent OOM contexts
func (occ *OOMContextCollector) GetRecentOOMs() map[uint32]*OOMContext {
	occ.mutex.RLock()
	defer occ.mutex.RUnlock()

	result := make(map[uint32]*OOMContext)
	for pid, ctx := range occ.recentOOMs {
		result[pid] = ctx
	}
	return result
}

// getOOMScore reads the OOM score for a process
func getOOMScore(pid uint32) int {
	// This would normally read from /proc/PID/oom_score
	// but the process is likely already gone, so return -1
	return -1
}

// GetMemoryPressureCategory categorizes memory pressure levels for metrics
func GetMemoryPressureCategory(level string) string {
	switch level {
	case "none":
		return "none"
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high", "critical":
		return "high"
	default:
		return "unknown"
	}
}

// GetMemoryUsageCategory categorizes memory usage for metrics
func GetMemoryUsageCategory(usagePercent float64) string {
	if usagePercent < 50 {
		return "low"
	} else if usagePercent < 80 {
		return "medium"
	} else if usagePercent < 95 {
		return "high"
	}
	return "critical"
}
