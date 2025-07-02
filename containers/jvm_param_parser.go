package containers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/coroot/coroot-node-agent/jvm"
	"k8s.io/klog/v2"
)

type JVMParams struct {
	JavaMaxHeapSize             string // heap size as string (e.g., "1073741824")
	JavaInitialHeapSize         string // heap size as string (e.g., "268435456")
	JavaMaxHeapAsPercentage     string // percentage value as string (e.g., "75.0")
	JavaInitialHeapAsPercentage string // percentage value as string (e.g., "25.0")
	MinRAMPercentage            string // minimum RAM percentage as string (e.g., "50.0")
	GCType                      string // garbage collector type (e.g., G1GC, SerialGC, ParallelGC, etc.)
}

func ParseJVMParams(pid uint32) JVMParams {
	// Get VM flags directly from the running JVM
	vmFlags, err := jvm.GetVMFlags(pid)
	if err != nil {
		klog.Warningf("Failed to get VM flags for PID %d (only HotSpot JVMs supported): %v", pid, err)
		return JVMParams{GCType: "Unknown"}
	}

	if strings.TrimSpace(vmFlags) == "" {
		klog.Warningf("Empty VM flags output for PID %d", pid)
		return JVMParams{GCType: "Unknown"}
	}

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
//
// Precedence rules for RAM parameters:
// - When both percentage and fraction parameters are present, percentage always takes precedence
// - MaxRAMPercentage takes precedence over MaxRAMFraction
// - InitialRAMPercentage takes precedence over InitialRAMFraction
// - MinRAMPercentage takes precedence over MinRAMFraction
// - This behavior is consistent with JVM behavior where -XX:MaxRAMPercentage effectively ignores -XX:MaxRAMFraction
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
					params.JavaMaxHeapSize = value
				}
			} else if strings.Contains(flag, "MinHeapSize=") {
				if value := extractFlagValue(flag, "MinHeapSize"); value != "" {
					params.JavaInitialHeapSize = value
				}
			} else if strings.Contains(flag, "InitialHeapSize=") {
				if value := extractFlagValue(flag, "InitialHeapSize"); value != "" {
					params.JavaInitialHeapSize = value
				}
			} else if strings.Contains(flag, "MaxRAMPercentage=") {
				if value := extractFlagValue(flag, "MaxRAMPercentage"); value != "" {
					params.JavaMaxHeapAsPercentage = value
				}
			} else if strings.Contains(flag, "InitialRAMPercentage=") {
				if value := extractFlagValue(flag, "InitialRAMPercentage"); value != "" {
					params.JavaInitialHeapAsPercentage = value
				}
			} else if strings.Contains(flag, "MinRAMPercentage=") {
				if value := extractFlagValue(flag, "MinRAMPercentage"); value != "" {
					params.MinRAMPercentage = value
				}
			} else if strings.Contains(flag, "MaxRAMFraction=") {
				// Convert fraction to percentage if percentage not already set
				// NOTE: MaxRAMPercentage takes precedence over MaxRAMFraction when both exist
				if params.JavaMaxHeapAsPercentage == "" {
					if value := extractFlagValue(flag, "MaxRAMFraction"); value != "" {
						if fraction, err := strconv.ParseFloat(value, 64); err == nil && fraction > 0 {
							params.JavaMaxHeapAsPercentage = fmt.Sprintf("%.1f", 100.0/fraction)
						}
					}
				}
			} else if strings.Contains(flag, "InitialRAMFraction=") {
				// Convert fraction to percentage if percentage not already set
				// NOTE: InitialRAMPercentage takes precedence over InitialRAMFraction when both exist
				if params.JavaInitialHeapAsPercentage == "" {
					if value := extractFlagValue(flag, "InitialRAMFraction"); value != "" {
						if fraction, err := strconv.ParseFloat(value, 64); err == nil && fraction > 0 {
							params.JavaInitialHeapAsPercentage = fmt.Sprintf("%.1f", 100.0/fraction)
						}
					}
				}
			} else if strings.Contains(flag, "MinRAMFraction=") {
				// Convert fraction to percentage if percentage not already set
				// NOTE: MinRAMPercentage takes precedence over MinRAMFraction when both exist
				if params.MinRAMPercentage == "" {
					if value := extractFlagValue(flag, "MinRAMFraction"); value != "" {
						if fraction, err := strconv.ParseFloat(value, 64); err == nil && fraction > 0 {
							params.MinRAMPercentage = fmt.Sprintf("%.1f", 100.0/fraction)
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
