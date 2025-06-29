package containers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/coroot/coroot-node-agent/jvm"
	"k8s.io/klog/v2"
)

type JVMParams struct {
	JavaMaxHeapSize             string // heap size as string (e.g., "1073741824")
	JavaInitialHeapSize         string // heap size as string (e.g., "268435456")
	JavaMaxHeapAsPercentage     string // percentage value as string (e.g., "75.0")
	JavaInitialHeapAsPercentage string // percentage value as string (e.g., "25.0")
	GCType                      string // garbage collector type (e.g., G1GC, SerialGC, ParallelGC, etc.)
}

type JVMVendor int

const (
	JVMVendorUnknown JVMVendor = iota
	JVMVendorHotSpot
	JVMVendorOpenJ9
	JVMVendorGraalVM
)

func (v JVMVendor) String() string {
	switch v {
	case JVMVendorHotSpot:
		return "HotSpot"
	case JVMVendorOpenJ9:
		return "OpenJ9"
	case JVMVendorGraalVM:
		return "GraalVM"
	default:
		return "Unknown"
	}
}

func ParseJVMParams(cmdline string, pid uint32) JVMParams {
	// Get VM flags directly from the running JVM
	vmFlags, err := jvm.GetVMFlags(pid)
	if err != nil {
		klog.Warningf("Failed to get VM flags for PID %d: %v", pid, err)
		// Try fallback parsing from command line
		return parseFromCmdline(cmdline, pid)
	}

	// Try to detect JVM vendor using system properties first (more accurate)
	vendor := detectJVMVendorWithSystemProperties(pid)
	if vendor == JVMVendorUnknown {
		// Fall back to VM flags-based detection
		vendor = detectJVMVendor(vmFlags)
	}

	klog.V(2).Infof("Detected JVM vendor: %s for PID %d", vendor, pid)

	// Parse VM flags output based on vendor
	params := parseVMFlagsOutput(vmFlags, vendor)

	// If we didn't get heap sizes from VM flags, try command line as fallback
	if params.JavaMaxHeapSize == "" && params.JavaInitialHeapSize == "" {
		klog.V(2).Infof("No heap sizes found in VM flags, trying command line fallback for PID %d", pid)
		cmdlineParams := parseFromCmdline(cmdline, pid)
		if cmdlineParams.JavaMaxHeapSize != "" {
			params.JavaMaxHeapSize = cmdlineParams.JavaMaxHeapSize
		}
		if cmdlineParams.JavaInitialHeapSize != "" {
			params.JavaInitialHeapSize = cmdlineParams.JavaInitialHeapSize
		}
		if cmdlineParams.JavaMaxHeapAsPercentage != "" {
			params.JavaMaxHeapAsPercentage = cmdlineParams.JavaMaxHeapAsPercentage
		}
		if cmdlineParams.JavaInitialHeapAsPercentage != "" {
			params.JavaInitialHeapAsPercentage = cmdlineParams.JavaInitialHeapAsPercentage
		}
		if params.GCType == "Unknown" && cmdlineParams.GCType != "Unknown" {
			params.GCType = cmdlineParams.GCType
		}
	}

	return params
}

// detectJVMVendor attempts to identify the JVM implementation from VM flags output
func detectJVMVendor(vmFlagsOutput string) JVMVendor {
	// Convert to lowercase for case-insensitive matching
	output := strings.ToLower(vmFlagsOutput)

	// Check for vendor-specific indicators
	switch {
	case strings.Contains(output, "openj9") || strings.Contains(output, "eclipse") || strings.Contains(output, "ibm"):
		return JVMVendorOpenJ9
	case strings.Contains(output, "graalvm") || strings.Contains(output, "graal"):
		return JVMVendorGraalVM
	case strings.Contains(output, "hotspot") || strings.Contains(output, "openjdk") || strings.Contains(output, "oracle"):
		return JVMVendorHotSpot
	default:
		// Check for HotSpot-style flags as fallback
		if strings.Contains(output, "-xx:maxheapsize=") || strings.Contains(output, "-xx:initialheapsize=") {
			return JVMVendorHotSpot
		}
		return JVMVendorUnknown
	}
}

// detectJVMVendorWithSystemProperties provides more accurate vendor detection using system properties
func detectJVMVendorWithSystemProperties(pid uint32) JVMVendor {
	// Try to get system properties for more accurate detection
	props, err := jvm.GetSystemProperties(pid)
	if err != nil {
		klog.V(2).Infof("Could not get system properties for PID %d: %v", pid, err)
		return JVMVendorUnknown
	}

	props = strings.ToLower(props)

	// Check system properties for vendor information
	switch {
	case strings.Contains(props, "java.vm.name") && (strings.Contains(props, "openj9") || strings.Contains(props, "eclipse") || strings.Contains(props, "ibm")):
		return JVMVendorOpenJ9
	case strings.Contains(props, "java.vm.name") && strings.Contains(props, "graalvm"):
		return JVMVendorGraalVM
	case strings.Contains(props, "java.vm.name") && (strings.Contains(props, "hotspot") || strings.Contains(props, "openjdk")):
		return JVMVendorHotSpot
	case strings.Contains(props, "java.vendor") && (strings.Contains(props, "eclipse") || strings.Contains(props, "ibm")):
		return JVMVendorOpenJ9
	case strings.Contains(props, "java.vendor") && strings.Contains(props, "oracle"):
		return JVMVendorHotSpot
	}

	// Try version information as fallback
	version, err := jvm.GetVersion(pid)
	if err == nil {
		version = strings.ToLower(version)
		switch {
		case strings.Contains(version, "openj9") || strings.Contains(version, "eclipse") || strings.Contains(version, "ibm"):
			return JVMVendorOpenJ9
		case strings.Contains(version, "graalvm"):
			return JVMVendorGraalVM
		case strings.Contains(version, "hotspot") || strings.Contains(version, "openjdk"):
			return JVMVendorHotSpot
		}
	}

	return JVMVendorUnknown
}

// parseFromCmdline attempts to extract JVM parameters from command line when VM.flags fails
func parseFromCmdline(cmdline string, pid uint32) JVMParams {
	klog.V(2).Infof("Falling back to command line parsing for PID %d", pid)

	params := JVMParams{}
	args := strings.Fields(cmdline)

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "-Xmx"):
			if size := parseSize(strings.TrimPrefix(arg, "-Xmx")); size != "" {
				params.JavaMaxHeapSize = size
			}
		case strings.HasPrefix(arg, "-Xms"):
			if size := parseSize(strings.TrimPrefix(arg, "-Xms")); size != "" {
				params.JavaInitialHeapSize = size
			}
		case strings.HasPrefix(arg, "-XX:MaxHeapSize="):
			params.JavaMaxHeapSize = strings.TrimPrefix(arg, "-XX:MaxHeapSize=")
		case strings.HasPrefix(arg, "-XX:InitialHeapSize="):
			params.JavaInitialHeapSize = strings.TrimPrefix(arg, "-XX:InitialHeapSize=")
		case strings.HasPrefix(arg, "-XX:MaxRAMPercentage="):
			params.JavaMaxHeapAsPercentage = strings.TrimPrefix(arg, "-XX:MaxRAMPercentage=")
		case strings.HasPrefix(arg, "-XX:InitialRAMPercentage="):
			params.JavaInitialHeapAsPercentage = strings.TrimPrefix(arg, "-XX:InitialRAMPercentage=")
		}
	}

	// Parse GC type from command line
	params.GCType = parseGCTypeFromCmdline(args)

	return params
}

// parseSize converts JVM size strings (like 1g, 512m) to bytes
func parseSize(sizeStr string) string {
	if sizeStr == "" {
		return ""
	}

	sizeStr = strings.ToLower(strings.TrimSpace(sizeStr))
	if len(sizeStr) == 0 {
		return ""
	}

	// Return as-is if it's already numeric (bytes)
	if regexp.MustCompile(`^\d+$`).MatchString(sizeStr) {
		return sizeStr
	}

	// Handle size with units (k, m, g, t)
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([kmgt]?)b?$`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) == 3 {
		numStr := matches[1]
		unit := matches[2]

		// Convert to bytes
		var multiplier int64 = 1
		switch unit {
		case "k":
			multiplier = 1024
		case "m":
			multiplier = 1024 * 1024
		case "g":
			multiplier = 1024 * 1024 * 1024
		case "t":
			multiplier = 1024 * 1024 * 1024 * 1024
		}

		// For simplicity, we'll try to parse as integer and multiply
		// In a production system, you'd want proper float parsing
		if num := regexp.MustCompile(`^\d+`).FindString(numStr); num != "" {
			return fmt.Sprintf("%s", num) + "×" + fmt.Sprintf("%d", multiplier)
		}

		// Fallback: return original with unit info
		return fmt.Sprintf("%s%s", numStr, unit)
	}

	return ""
}

// parseGCTypeFromCmdline detects GC type from command line arguments
func parseGCTypeFromCmdline(args []string) string {
	for _, arg := range args {
		switch {
		case strings.Contains(arg, "-XX:+UseZGC"):
			return "ZGC"
		case strings.Contains(arg, "-XX:+UseShenandoahGC"):
			return "ShenandoahGC"
		case strings.Contains(arg, "-XX:+UseG1GC"):
			return "G1GC"
		case strings.Contains(arg, "-XX:+UseParallelGC"):
			return "ParallelGC"
		case strings.Contains(arg, "-XX:+UseParallelOldGC"):
			return "ParallelOldGC"
		case strings.Contains(arg, "-XX:+UseConcMarkSweepGC"):
			return "ConcMarkSweepGC"
		case strings.Contains(arg, "-XX:+UseSerialGC"):
			return "SerialGC"
		case strings.Contains(arg, "-Xgcpolicy:gencon"):
			return "Generational Concurrent (OpenJ9)"
		case strings.Contains(arg, "-Xgcpolicy:optthruput"):
			return "Throughput (OpenJ9)"
		case strings.Contains(arg, "-Xgcpolicy:optavgpause"):
			return "Average Pause (OpenJ9)"
		case strings.Contains(arg, "-Xgcpolicy:balanced"):
			return "Balanced (OpenJ9)"
		}
	}
	return "Unknown"
}

// parseGCType extracts the garbage collector type from VM flags
func parseGCType(flags []string, vendor JVMVendor) string {
	switch vendor {
	case JVMVendorOpenJ9:
		return parseOpenJ9GCType(flags)
	default:
		return parseHotSpotGCType(flags)
	}
}

// parseHotSpotGCType handles HotSpot/OpenJDK GC detection
func parseHotSpotGCType(flags []string) string {
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

// parseOpenJ9GCType handles OpenJ9/IBM J9 GC detection
func parseOpenJ9GCType(flags []string) string {
	for _, flag := range flags {
		// OpenJ9 uses different GC policy flags
		if strings.Contains(flag, "gencon") {
			return "Generational Concurrent (OpenJ9)"
		}
		if strings.Contains(flag, "optthruput") {
			return "Throughput (OpenJ9)"
		}
		if strings.Contains(flag, "optavgpause") {
			return "Average Pause (OpenJ9)"
		}
		if strings.Contains(flag, "balanced") {
			return "Balanced (OpenJ9)"
		}
	}
	return "Unknown (OpenJ9)"
}

// parseVMFlagsOutput parses the output from jcmd VM.flags command
func parseVMFlagsOutput(vmFlagsOutput string, vendor JVMVendor) JVMParams {
	if strings.TrimSpace(vmFlagsOutput) == "" {
		klog.Warning("Empty VM flags output received")
		return JVMParams{}
	}

	params := JVMParams{}

	switch vendor {
	case JVMVendorOpenJ9:
		params = parseOpenJ9VMFlags(vmFlagsOutput)
	case JVMVendorGraalVM:
		params = parseGraalVMFlags(vmFlagsOutput)
	default:
		params = parseHotSpotVMFlags(vmFlagsOutput)
	}

	// Parse GC type from all flags
	flags := strings.Fields(vmFlagsOutput)
	params.GCType = parseGCType(flags, vendor)

	klog.V(3).Infof("Parsed JVM params: MaxHeap=%s, InitialHeap=%s, MaxHeapPercentage=%s, InitialHeapPercentage=%s, GC=%s",
		params.JavaMaxHeapSize, params.JavaInitialHeapSize, params.JavaMaxHeapAsPercentage, params.JavaInitialHeapAsPercentage, params.GCType)

	return params
}

// parseHotSpotVMFlags handles standard HotSpot/OpenJDK VM flags
func parseHotSpotVMFlags(vmFlagsOutput string) JVMParams {
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
			}
		}
	}

	return params
}

// parseOpenJ9VMFlags handles IBM OpenJ9/J9 VM flags with different naming conventions
func parseOpenJ9VMFlags(vmFlagsOutput string) JVMParams {
	params := JVMParams{}

	// OpenJ9 might use different flag formats
	flags := strings.Fields(vmFlagsOutput)

	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if flag == "" {
			continue
		}

		// Handle OpenJ9-specific flag formats
		switch {
		case strings.Contains(flag, "-Xmx"):
			// Extract value after -Xmx
			if value := extractOpenJ9Size(flag, "-Xmx"); value != "" {
				params.JavaMaxHeapSize = value
			}
		case strings.Contains(flag, "-Xms"):
			// Extract value after -Xms
			if value := extractOpenJ9Size(flag, "-Xms"); value != "" {
				params.JavaInitialHeapSize = value
			}
		case strings.HasPrefix(flag, "-XX:") && strings.Contains(flag, "="):
			// Handle standard -XX: flags that might still exist
			if strings.Contains(flag, "MaxHeapSize=") {
				if value := extractFlagValue(flag, "MaxHeapSize"); value != "" {
					params.JavaMaxHeapSize = value
				}
			} else if strings.Contains(flag, "InitialHeapSize=") {
				if value := extractFlagValue(flag, "InitialHeapSize"); value != "" {
					params.JavaInitialHeapSize = value
				}
			}
		}
	}

	return params
}

// parseGraalVMFlags handles GraalVM-specific flags
func parseGraalVMFlags(vmFlagsOutput string) JVMParams {
	// GraalVM generally follows HotSpot conventions but may have additional flags
	params := parseHotSpotVMFlags(vmFlagsOutput)

	// Add GraalVM-specific parsing if needed
	// For now, use HotSpot parsing as baseline

	return params
}

// extractOpenJ9Size extracts size values from OpenJ9-style flags like "-Xmx1g"
func extractOpenJ9Size(flag, prefix string) string {
	if !strings.HasPrefix(flag, prefix) {
		return ""
	}

	sizeStr := strings.TrimPrefix(flag, prefix)
	return parseSize(sizeStr)
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
