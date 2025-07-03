package node

import (
	"bufio"
	"os"
	"path"
	"strconv"
	"strings"
)

// PressureStats represents pressure stall information for a resource
type PressureStats struct {
	Some PressureMetrics `json:"some"`
	Full PressureMetrics `json:"full"`
}

// PressureMetrics contains the pressure metrics
type PressureMetrics struct {
	Avg10  float64 `json:"avg10"`  // 10-second average
	Avg60  float64 `json:"avg60"`  // 60-second average
	Avg300 float64 `json:"avg300"` // 300-second average
	Total  uint64  `json:"total"`  // Total time in microseconds
}

// SystemPressure contains pressure information for all resources
type SystemPressure struct {
	Memory PressureStats `json:"memory"`
	CPU    PressureStats `json:"cpu"`
	IO     PressureStats `json:"io"`
}

// GetSystemPressure reads pressure stall information from /proc/pressure
func GetSystemPressure(procRoot string) (*SystemPressure, error) {
	pressure := &SystemPressure{}

	// Read memory pressure
	if stats, err := readPressureFile(path.Join(procRoot, "pressure", "memory")); err == nil {
		pressure.Memory = *stats
	}

	// Read CPU pressure
	if stats, err := readPressureFile(path.Join(procRoot, "pressure", "cpu")); err == nil {
		pressure.CPU = *stats
	}

	// Read IO pressure
	if stats, err := readPressureFile(path.Join(procRoot, "pressure", "io")); err == nil {
		pressure.IO = *stats
	}

	return pressure, nil
}

// readPressureFile parses a single pressure file
func readPressureFile(filename string) (*PressureStats, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats := &PressureStats{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}

		var metrics *PressureMetrics
		switch parts[0] {
		case "some":
			metrics = &stats.Some
		case "full":
			metrics = &stats.Full
		default:
			continue
		}

		// Parse avg10=X.XX avg60=X.XX avg300=X.XX total=XXXXX
		for i := 1; i < len(parts); i++ {
			kv := strings.Split(parts[i], "=")
			if len(kv) != 2 {
				continue
			}

			switch kv[0] {
			case "avg10":
				if val, err := strconv.ParseFloat(kv[1], 64); err == nil {
					metrics.Avg10 = val
				}
			case "avg60":
				if val, err := strconv.ParseFloat(kv[1], 64); err == nil {
					metrics.Avg60 = val
				}
			case "avg300":
				if val, err := strconv.ParseFloat(kv[1], 64); err == nil {
					metrics.Avg300 = val
				}
			case "total":
				if val, err := strconv.ParseUint(kv[1], 10, 64); err == nil {
					metrics.Total = val
				}
			}
		}
	}

	return stats, scanner.Err()
}

// IsMemoryPressureHigh determines if memory pressure is considered high
func (p *SystemPressure) IsMemoryPressureHigh() bool {
	// Consider memory pressure high if:
	// - 10s average > 10% OR
	// - 60s average > 5% OR
	// - 300s average > 1%
	return p.Memory.Some.Avg10 > 10.0 ||
		p.Memory.Some.Avg60 > 5.0 ||
		p.Memory.Some.Avg300 > 1.0
}

// GetMemoryPressureLevel returns a string indicating the pressure level
func (p *SystemPressure) GetMemoryPressureLevel() string {
	if p.Memory.Some.Avg10 > 50.0 || p.Memory.Full.Avg10 > 10.0 {
		return "critical"
	} else if p.Memory.Some.Avg10 > 20.0 || p.Memory.Full.Avg10 > 1.0 {
		return "high"
	} else if p.Memory.Some.Avg10 > 10.0 || p.Memory.Some.Avg60 > 5.0 {
		return "medium"
	} else if p.Memory.Some.Avg10 > 0.0 || p.Memory.Some.Avg60 > 0.0 {
		return "low"
	}
	return "none"
}
