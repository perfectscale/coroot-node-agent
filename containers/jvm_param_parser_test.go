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
			name:  "VM flags with MaxRAMFraction - converts to percentage",
			input: "-XX:MaxRAMFraction=4 -XX:+UseSerialGC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "25.0", // 100/4 = 25.0
				JavaInitialHeapAsPercentage: "",
				MinRAMPercentage:            "",
				GCType:                      "SerialGC",
			},
		},
		{
			name:  "VM flags with InitialRAMFraction - converts to percentage",
			input: "-XX:InitialRAMFraction=8 -XX:+UseG1GC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "",
				JavaInitialHeapAsPercentage: "12.5", // 100/8 = 12.5
				MinRAMPercentage:            "",
				GCType:                      "G1GC",
			},
		},
		{
			name:  "VM flags with MinRAMFraction - converts to percentage",
			input: "-XX:MinRAMFraction=2 -XX:+UseParallelGC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "",
				JavaInitialHeapAsPercentage: "",
				MinRAMPercentage:            "50.0", // 100/2 = 50.0
				GCType:                      "ParallelGC",
			},
		},
		{
			name:  "VM flags with all fraction parameters",
			input: "-XX:MaxRAMFraction=4 -XX:InitialRAMFraction=8 -XX:MinRAMFraction=2 -XX:+UseZGC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "25.0", // 100/4 = 25.0
				JavaInitialHeapAsPercentage: "12.5", // 100/8 = 12.5
				MinRAMPercentage:            "50.0", // 100/2 = 50.0
				GCType:                      "ZGC",
			},
		},
		{
			name:  "Percentage parameters take precedence over fraction parameters",
			input: "-XX:MaxRAMFraction=4 -XX:MaxRAMPercentage=80.0 -XX:InitialRAMFraction=8 -XX:InitialRAMPercentage=30.0 -XX:MinRAMFraction=2 -XX:MinRAMPercentage=60.0 -XX:+UseG1GC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "80.0", // Percentage takes precedence
				JavaInitialHeapAsPercentage: "30.0", // Percentage takes precedence
				MinRAMPercentage:            "60.0", // Percentage takes precedence
				GCType:                      "G1GC",
			},
		},
		{
			name:  "Percentage parameters take precedence - fraction after percentage",
			input: "-XX:MaxRAMPercentage=75.0 -XX:MaxRAMFraction=4 -XX:InitialRAMPercentage=20.0 -XX:InitialRAMFraction=8 -XX:MinRAMPercentage=45.0 -XX:MinRAMFraction=2 -XX:+UseG1GC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "75.0", // Percentage takes precedence even when fraction comes after
				JavaInitialHeapAsPercentage: "20.0", // Percentage takes precedence even when fraction comes after
				MinRAMPercentage:            "45.0", // Percentage takes precedence even when fraction comes after
				GCType:                      "G1GC",
			},
		},
		{
			name:  "Mixed heap size and fraction parameters",
			input: "-XX:MaxHeapSize=2147483648 -XX:InitialRAMFraction=8 -XX:MinRAMPercentage=40.0 -XX:+UseConcMarkSweepGC",
			expected: JVMParams{
				JavaMaxHeapSize:             "2147483648",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "",
				JavaInitialHeapAsPercentage: "12.5", // 100/8 = 12.5
				MinRAMPercentage:            "40.0",
				GCType:                      "ConcMarkSweepGC",
			},
		},
		{
			name:  "Invalid fraction parameters are ignored",
			input: "-XX:MaxRAMFraction=0 -XX:InitialRAMFraction=invalid -XX:MinRAMFraction=-1 -XX:+UseParallelGC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "", // 0 is invalid, ignored
				JavaInitialHeapAsPercentage: "", // invalid string ignored
				MinRAMPercentage:            "", // negative is invalid, ignored
				GCType:                      "ParallelGC",
			},
		},
		{
			name:  "Decimal fraction parameters",
			input: "-XX:MaxRAMFraction=3.5 -XX:InitialRAMFraction=6.25 -XX:+UseG1GC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "28.6", // 100/3.5 = 28.571... rounded to 28.6
				JavaInitialHeapAsPercentage: "16.0", // 100/6.25 = 16.0
				MinRAMPercentage:            "",
				GCType:                      "G1GC",
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
		{
			name:  "Complex realistic VM flags",
			input: "-XX:MaxHeapSize=4294967296 -XX:InitialHeapSize=268435456 -XX:MaxRAMPercentage=80.0 -XX:InitialRAMPercentage=6.25 -XX:MinRAMPercentage=50.0 -XX:MaxRAMFraction=2 -XX:+UseG1GC -XX:MaxGCPauseMillis=200 -XX:G1HeapRegionSize=16777216",
			expected: JVMParams{
				JavaMaxHeapSize:             "4294967296",
				JavaInitialHeapSize:         "268435456",
				JavaMaxHeapAsPercentage:     "80.0", // Percentage takes precedence over fraction
				JavaInitialHeapAsPercentage: "6.25",
				MinRAMPercentage:            "50.0",
				GCType:                      "G1GC",
			},
		},
		{
			name:  "Multiple fraction and percentage parameters mixed order",
			input: "-XX:MaxRAMFraction=3 -XX:InitialRAMPercentage=15.0 -XX:MinRAMFraction=4 -XX:MaxRAMPercentage=70.0 -XX:InitialRAMFraction=10 -XX:MinRAMPercentage=35.0 -XX:+UseParallelGC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "70.0", // Percentage always wins
				JavaInitialHeapAsPercentage: "15.0", // Percentage always wins
				MinRAMPercentage:            "35.0", // Percentage always wins
				GCType:                      "ParallelGC",
			},
		},
		{
			name:  "Only one parameter has both fraction and percentage",
			input: "-XX:MaxRAMFraction=2 -XX:MaxRAMPercentage=90.0 -XX:InitialRAMFraction=8 -XX:MinRAMPercentage=40.0 -XX:+UseG1GC",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "90.0", // Percentage takes precedence
				JavaInitialHeapAsPercentage: "12.5", // Only fraction provided, converts to percentage
				MinRAMPercentage:            "40.0", // Only percentage provided
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
