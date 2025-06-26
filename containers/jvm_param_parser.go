package containers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/coroot/coroot-node-agent/jvm"
)

type JVMParams struct {
	JavaMaxHeapSize             float64 // in bytes, -1 if using percentage
	JavaInitialHeapSize         float64 // in bytes, -1 if using percentage
	JavaMaxHeapAsPercentage     float64 // percentage value, 0 if not set
	JavaInitialHeapAsPercentage float64 // percentage value, 0 if not set
	GCType                      string  // garbage collector type (e.g., G1GC, SerialGC, ParallelGC, etc.)
}

// parseMemorySize converts memory size strings like "2g", "512m", "1024" to bytes
func parseMemorySize(sizeStr string) (float64, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Extract numeric part and unit
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)([kmgKMG]?)$`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid memory size format: %s", sizeStr)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %s", matches[1])
	}

	unit := strings.ToLower(matches[2])
	switch unit {
	case "k":
		return value * 1024, nil
	case "m":
		return value * 1024 * 1024, nil
	case "g":
		return value * 1024 * 1024 * 1024, nil
	case "":
		return value, nil // bytes
	default:
		return 0, fmt.Errorf("unknown memory unit: %s", unit)
	}
}

func ParseJVMParams(cmdline string, pid uint32) JVMParams {
	// Get VM flags directly from the running JVM
	vmFlags, err := jvm.GetVMFlags(pid)
	if err != nil {
		// Return empty params if we can't get VM flags
		return JVMParams{}
	}

	// Parse VM flags output
	return parseVMFlagsOutput(vmFlags)
}

// parseGCType extracts the garbage collector type from VM flags
func parseGCType(flags []string) string {
	// GC flags in order of precedence (newer/more specific GCs first)
	gcFlags := []struct {
		flag   string
		gcType string
	}{
		{"+UseZGC", "ZGC"},
		{"+UseShenandoahGC", "ShenandoahGC"},
		{"+UseG1GC", "G1GC"},
		{"+UseParallelGC", "ParallelGC"},
		{"+UseParallelOldGC", "ParallelOldGC"},
		{"+UseConcMarkSweepGC", "ConcMarkSweepGC"},
		{"+UseSerialGC", "SerialGC"},
	}

	// Look for enabled GC flags (last one wins if multiple are specified)
	var detectedGC string
	for _, flag := range flags {
		for _, gc := range gcFlags {
			if strings.Contains(flag, gc.flag) {
				detectedGC = gc.gcType
			}
		}
	}

	// If no explicit GC flag found, try to infer from other flags
	if detectedGC == "" {
		for _, flag := range flags {
			if strings.Contains(flag, "G1") {
				return "G1GC"
			}
			if strings.Contains(flag, "Parallel") && !strings.Contains(flag, "-UseParallelGC") {
				return "ParallelGC"
			}
			if strings.Contains(flag, "ConcMarkSweep") || strings.Contains(flag, "CMS") {
				return "ConcMarkSweepGC"
			}
			if strings.Contains(flag, "Serial") && !strings.Contains(flag, "-UseSerialGC") {
				return "SerialGC"
			}
		}
	}

	// Default to unknown if no GC type can be determined
	if detectedGC == "" {
		return "Unknown"
	}

	return detectedGC
}

// parseVMFlagsOutput parses the output from jcmd VM.flags command
func parseVMFlagsOutput(vmFlagsOutput string) JVMParams {
	params := JVMParams{}

	// Split the output by spaces to get individual flags
	flags := strings.Fields(vmFlagsOutput)

	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if flag == "" {
			continue
		}

		// Parse VM flags in format: -XX:MaxHeapSize=2147483648
		if strings.HasPrefix(flag, "-XX:") {
			// Parse specific flags we care about
			if strings.Contains(flag, "MaxHeapSize=") {
				if value := extractFlagValue(flag, "MaxHeapSize"); value != "" {
					if size, err := strconv.ParseFloat(value, 64); err == nil {
						params.JavaMaxHeapSize = size
					}
				}
			} else if strings.Contains(flag, "MinHeapSize=") {
				if value := extractFlagValue(flag, "MinHeapSize"); value != "" {
					if size, err := strconv.ParseFloat(value, 64); err == nil {
						params.JavaInitialHeapSize = size
					}
				}
			} else if strings.Contains(flag, "InitialHeapSize=") {
				if value := extractFlagValue(flag, "InitialHeapSize"); value != "" {
					if size, err := strconv.ParseFloat(value, 64); err == nil {
						params.JavaInitialHeapSize = size
					}
				}
			} else if strings.Contains(flag, "MaxRAMPercentage=") {
				if value := extractFlagValue(flag, "MaxRAMPercentage"); value != "" {
					if percentage, err := strconv.ParseFloat(value, 64); err == nil {
						params.JavaMaxHeapAsPercentage = percentage
						if params.JavaMaxHeapSize == 0 {
							params.JavaMaxHeapSize = -1 // Use percentage
						}
					}
				}
			} else if strings.Contains(flag, "InitialRAMPercentage=") {
				if value := extractFlagValue(flag, "InitialRAMPercentage"); value != "" {
					if percentage, err := strconv.ParseFloat(value, 64); err == nil {
						params.JavaInitialHeapAsPercentage = percentage
						if params.JavaInitialHeapSize == 0 {
							params.JavaInitialHeapSize = -1 // Use percentage
						}
					}
				}
			}
		}
	}

	// Parse GC type from all flags
	params.GCType = parseGCType(flags)

	return params
}

// extractFlagValue extracts the value from a VM flag like "-XX:MaxHeapSize=2147483648"
func extractFlagValue(line, flagName string) string {
	pattern := fmt.Sprintf(`-XX:%s=([^\s]+)`, flagName)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
