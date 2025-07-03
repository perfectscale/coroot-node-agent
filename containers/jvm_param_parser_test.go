package containers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVMFlagsOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected JVMParams
	}{
		{
			name:  "Basic VM flags with heap sizes",
			input: "-XX:MaxHeapSize=1073741824 -XX:InitialHeapSize=268435456 -XX:+UseG1GC",
			expected: JVMParams{
				JavaMaxHeapSize:             "1073741824",
				JavaInitialHeapSize:         "268435456",
				JavaMaxHeapAsPercentage:     "",
				JavaInitialHeapAsPercentage: "",
				MinRAMPercentage:            "",
				GCType:                      "G1GC",
			},
		},
		{
			name:  "VM flags with percentage parameters",
			input: "-XX:MaxRAMPercentage=75.0 -XX:InitialRAMPercentage=25.0 -XX:MinRAMPercentage=50.0 -XX:+UseParallelGC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "75.0",
				JavaInitialHeapAsPercentage: "25.0",
				MinRAMPercentage:            "50.0",
				GCType:                      "ParallelGC",
			},
		},
		{
			name:  "No GC flag - defaults to Unknown",
			input: "-XX:MaxRAMPercentage=75.0",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "75.0",
				JavaInitialHeapAsPercentage: "",
				MinRAMPercentage:            "",
				GCType:                      "Unknown",
			},
		},
		{
			name:  "Multiple GC flags - last one wins",
			input: "-XX:+UseSerialGC -XX:+UseParallelGC -XX:+UseG1GC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "",
				JavaInitialHeapAsPercentage: "",
				MinRAMPercentage:            "",
				GCType:                      "G1GC",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVMFlagsOutput(tt.input)
			assert.Equal(t, tt.expected.JavaMaxHeapSize, result.JavaMaxHeapSize, "JavaMaxHeapSize mismatch")
			assert.Equal(t, tt.expected.JavaInitialHeapSize, result.JavaInitialHeapSize, "JavaInitialHeapSize mismatch")
			assert.Equal(t, tt.expected.JavaMaxHeapAsPercentage, result.JavaMaxHeapAsPercentage, "JavaMaxHeapAsPercentage mismatch")
			assert.Equal(t, tt.expected.JavaInitialHeapAsPercentage, result.JavaInitialHeapAsPercentage, "JavaInitialHeapAsPercentage mismatch")
			assert.Equal(t, tt.expected.MinRAMPercentage, result.MinRAMPercentage, "MinRAMPercentage mismatch")
			assert.Equal(t, tt.expected.GCType, result.GCType, "GCType mismatch")
		})
	}
}

func TestParseGCType(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		expected string
	}{
		{
			name:     "G1GC",
			flags:    []string{"-XX:+UseG1GC"},
			expected: "G1GC",
		},
		{
			name:     "ParallelGC",
			flags:    []string{"-XX:+UseParallelGC"},
			expected: "ParallelGC",
		},
		{
			name:     "SerialGC",
			flags:    []string{"-XX:+UseSerialGC"},
			expected: "SerialGC",
		},
		{
			name:     "ZGC",
			flags:    []string{"-XX:+UseZGC"},
			expected: "ZGC",
		},
		{
			name:     "ShenandoahGC",
			flags:    []string{"-XX:+UseShenandoahGC"},
			expected: "ShenandoahGC",
		},
		{
			name:     "ConcMarkSweepGC",
			flags:    []string{"-XX:+UseConcMarkSweepGC"},
			expected: "ConcMarkSweepGC",
		},
		{
			name:     "ParallelOldGC",
			flags:    []string{"-XX:+UseParallelOldGC"},
			expected: "ParallelOldGC",
		},
		{
			name:     "Multiple GC flags - last wins",
			flags:    []string{"-XX:+UseSerialGC", "-XX:+UseParallelGC", "-XX:+UseG1GC"},
			expected: "G1GC",
		},
		{
			name:     "No GC flags",
			flags:    []string{"-XX:MaxHeapSize=1g"},
			expected: "Unknown",
		},
		{
			name:     "Inferred from G1 in other flags",
			flags:    []string{"-XX:G1HeapRegionSize=16m"},
			expected: "G1GC",
		},
		{
			name:     "Inferred from Parallel in other flags",
			flags:    []string{"-XX:ParallelGCThreads=4"},
			expected: "ParallelGC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGCType(tt.flags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFlagValue(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		flagName string
		expected string
	}{
		{
			name:     "Simple flag extraction",
			line:     "-XX:MaxHeapSize=1073741824",
			flagName: "MaxHeapSize",
			expected: "1073741824",
		},
		{
			name:     "Flag with decimal value",
			line:     "-XX:MaxRAMPercentage=75.5",
			flagName: "MaxRAMPercentage",
			expected: "75.5",
		},
		{
			name:     "Flag not found",
			line:     "-XX:MaxHeapSize=1073741824",
			flagName: "MinHeapSize",
			expected: "",
		},
		{
			name:     "Flag with complex value",
			line:     "-XX:G1HeapRegionSize=16777216",
			flagName: "G1HeapRegionSize",
			expected: "16777216",
		},
		{
			name:     "Empty flag value",
			line:     "-XX:SomeFlag=",
			flagName: "SomeFlag",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFlagValue(tt.line, tt.flagName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
