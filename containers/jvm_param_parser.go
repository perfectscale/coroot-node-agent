package containers

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/coroot/coroot-node-agent/proc"
)

type JVMParams struct {
	JavaMaxHeapSize             float64 // in bytes, -1 if using percentage
	JavaInitialHeapSize         float64 // in bytes, -1 if using percentage
	JavaMaxHeapAsPercentage     float64 // percentage value, 0 if not set
	JavaInitialHeapAsPercentage float64 // percentage value, 0 if not set
	XXOptions                   string  // all other XX options as comma-separated string
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

// parsePercentage extracts percentage value from XX options like "MaxRAMPercentage=75.0"
// Uses the last occurrence to match real JVM behavior where later parameters override earlier ones
func parsePercentage(xxOptions, optionName string) float64 {
	pattern := fmt.Sprintf(`-XX:%s=([0-9]+(?:\.[0-9]+)?)`, optionName)
	re := regexp.MustCompile(pattern)
	allMatches := re.FindAllStringSubmatch(xxOptions, -1)
	if len(allMatches) > 0 {
		// Take the last match (rightmost parameter wins)
		lastMatch := allMatches[len(allMatches)-1]
		if len(lastMatch) > 1 {
			if value, err := strconv.ParseFloat(lastMatch[1], 64); err == nil {
				return value
			}
		}
	}
	return 0
}

func readEnviron(pid uint32) map[string]string {
	env := make(map[string]string)
	f, err := os.Open(proc.Path(pid, "environ"))
	if err != nil {
		return env
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString(0)
		if err != nil {
			break
		}
		// Remove the null terminator
		line = strings.TrimSuffix(line, "\x00")
		kv := strings.SplitN(line, "=", 2)
		if len(kv) == 2 {
			env[kv[0]] = kv[1]
		}
	}
	return env
}

func parseJVMParamsFromString(input string) JVMParams {
	params := JVMParams{}

	// Enhanced parsing: Handle both -Xmx and -XX:MaxHeapSize as equivalent
	// We'll find ALL max heap parameters (both -Xmx and -XX:MaxHeapSize) and use the rightmost one
	var maxHeapMatches []struct {
		value string
		pos   int // position in string for precedence
	}

	// Parse -Xmx parameters
	xmxRegex := regexp.MustCompile(`(?:^|\s)-Xmx(\d+[kmgKMG]?)`)
	xmxAllMatches := xmxRegex.FindAllStringSubmatch(input, -1)
	xmxIndices := xmxRegex.FindAllStringIndex(input, -1)
	for i, match := range xmxAllMatches {
		if len(match) > 1 {
			maxHeapMatches = append(maxHeapMatches, struct {
				value string
				pos   int
			}{match[1], xmxIndices[i][0]})
		}
	}

	// Parse -XX:MaxHeapSize parameters
	xxMaxHeapRegex := regexp.MustCompile(`(?:^|\s)-XX:MaxHeapSize=(\d+[kmgKMG]?)`)
	xxMaxHeapAllMatches := xxMaxHeapRegex.FindAllStringSubmatch(input, -1)
	xxMaxHeapIndices := xxMaxHeapRegex.FindAllStringIndex(input, -1)
	for i, match := range xxMaxHeapAllMatches {
		if len(match) > 1 {
			maxHeapMatches = append(maxHeapMatches, struct {
				value string
				pos   int
			}{match[1], xxMaxHeapIndices[i][0]})
		}
	}

	// Find the rightmost (latest) max heap parameter
	if len(maxHeapMatches) > 0 {
		var rightmostValue string
		rightmostPos := -1
		for _, match := range maxHeapMatches {
			if match.pos > rightmostPos {
				rightmostPos = match.pos
				rightmostValue = match.value
			}
		}
		if size, err := parseMemorySize(rightmostValue); err == nil {
			params.JavaMaxHeapSize = size
		}
	}

	// Enhanced parsing: Handle both -Xms and -XX:MinHeapSize
	// Note: -Xms sets initial heap, -XX:MinHeapSize sets minimum heap (different semantics)
	// but for monitoring purposes, we'll treat them as related to initial heap sizing
	var initialHeapMatches []struct {
		value string
		pos   int
	}

	// Parse -Xms parameters
	xmsRegex := regexp.MustCompile(`(?:^|\s)-Xms(\d+[kmgKMG]?)`)
	xmsAllMatches := xmsRegex.FindAllStringSubmatch(input, -1)
	xmsIndices := xmsRegex.FindAllStringIndex(input, -1)
	for i, match := range xmsAllMatches {
		if len(match) > 1 {
			initialHeapMatches = append(initialHeapMatches, struct {
				value string
				pos   int
			}{match[1], xmsIndices[i][0]})
		}
	}

	// Parse -XX:MinHeapSize parameters (treating as initial heap for monitoring)
	xxMinHeapRegex := regexp.MustCompile(`(?:^|\s)-XX:MinHeapSize=(\d+[kmgKMG]?)`)
	xxMinHeapAllMatches := xxMinHeapRegex.FindAllStringSubmatch(input, -1)
	xxMinHeapIndices := xxMinHeapRegex.FindAllStringIndex(input, -1)
	for i, match := range xxMinHeapAllMatches {
		if len(match) > 1 {
			initialHeapMatches = append(initialHeapMatches, struct {
				value string
				pos   int
			}{match[1], xxMinHeapIndices[i][0]})
		}
	}

	// Find the rightmost (latest) initial heap parameter
	if len(initialHeapMatches) > 0 {
		var rightmostValue string
		rightmostPos := -1
		for _, match := range initialHeapMatches {
			if match.pos > rightmostPos {
				rightmostPos = match.pos
				rightmostValue = match.value
			}
		}
		if size, err := parseMemorySize(rightmostValue); err == nil {
			params.JavaInitialHeapSize = size
		}
	}

	// Parse all -XX options (including the ones we just processed for completeness)
	xxRegex := regexp.MustCompile(`(?:^|\s)-XX:[+-]?[A-Za-z][A-Za-z0-9]*(?:=[^\s]+)?`)
	xxMatches := xxRegex.FindAllString(input, -1)

	var cleanMatches []string
	var xxOptionsString string

	if len(xxMatches) > 0 {
		// Clean up any leading spaces from matches
		for _, match := range xxMatches {
			cleanMatches = append(cleanMatches, strings.TrimSpace(match))
		}
		xxOptionsString = strings.Join(cleanMatches, ",")
		params.XXOptions = xxOptionsString
	}

	// Parse percentage-based memory settings from XX options
	params.JavaMaxHeapAsPercentage = parsePercentage(xxOptionsString, "MaxRAMPercentage")
	params.JavaInitialHeapAsPercentage = parsePercentage(xxOptionsString, "InitialRAMPercentage")

	// If using percentage-based settings, set corresponding heap sizes to -1
	if params.JavaMaxHeapAsPercentage > 0 {
		params.JavaMaxHeapSize = -1
	}
	if params.JavaInitialHeapAsPercentage > 0 {
		params.JavaInitialHeapSize = -1
	}

	return params
}

func parseJVMParams(cmdline string, pid uint32) JVMParams {
	// Real JVM behavior: Environment variables are processed FIRST (prepended to command line),
	// then command line parameters are processed AFTER (appended), so command line can override env vars.

	var combinedArgs strings.Builder
	env := readEnviron(pid)

	// Process environment variables in order of precedence (highest to lowest)
	// These get prepended to the argument list
	envVars := []string{
		"JAVA_TOOL_OPTIONS",
		"_JAVA_OPTIONS",
		"JDK_JAVA_OPTIONS",
		"IBM_JAVA_OPTIONS",
	}

	for _, envVar := range envVars {
		if envValue, exists := env[envVar]; exists && envValue != "" {
			if combinedArgs.Len() > 0 {
				combinedArgs.WriteString(" ")
			}
			combinedArgs.WriteString(envValue)
		}
	}

	// Append command line parameters (these can override environment variables)
	if cmdline != "" {
		if combinedArgs.Len() > 0 {
			combinedArgs.WriteString(" ")
		}
		combinedArgs.WriteString(cmdline)
	}

	// Parse the combined parameter string
	// Since command line parameters come last, they will override environment variables
	// when parseJVMParamsFromString finds multiple instances of the same parameter
	return parseJVMParamsFromString(combinedArgs.String())
}
