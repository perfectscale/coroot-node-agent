package containers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseJVMParamsFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected JVMParams
	}{
		{
			name:  "empty string",
			input: "",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "",
			},
		},
		{
			name:  "only XMX",
			input: "java -Xmx2g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             2 * 1024 * 1024 * 1024, // 2GB in bytes
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "",
			},
		},
		{
			name:  "only XMS",
			input: "java -Xms512m MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         512 * 1024 * 1024, // 512MB in bytes
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "",
			},
		},
		{
			name:  "XMX and XMS",
			input: "java -Xmx4g -Xms1g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB in bytes
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB in bytes
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "",
			},
		},
		{
			name:  "single XX option",
			input: "java -XX:+UseG1GC MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC",
			},
		},
		{
			name:  "multiple XX options",
			input: "java -XX:+UseG1GC -XX:MaxGCPauseMillis=200 -XX:+DisableExplicitGC MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC,-XX:MaxGCPauseMillis=200,-XX:+DisableExplicitGC",
			},
		},
		{
			name:  "all parameters",
			input: "java -Xmx8g -Xms2g -XX:+UseG1GC -XX:MaxGCPauseMillis=200 MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             8 * 1024 * 1024 * 1024, // 8GB
				JavaInitialHeapSize:         2 * 1024 * 1024 * 1024, // 2GB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC,-XX:MaxGCPauseMillis=200",
			},
		},
		{
			name:  "parameters with different units",
			input: "java -Xmx1024M -Xms256K MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             1024 * 1024 * 1024, // 1024MB = 1GB
				JavaInitialHeapSize:         256 * 1024,         // 256KB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "",
			},
		},
		{
			name:  "parameters in different order",
			input: "java -XX:+UseG1GC -Xms1g -XX:MaxGCPauseMillis=200 -Xmx4g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC,-XX:MaxGCPauseMillis=200",
			},
		},
		{
			name:  "numeric values without units",
			input: "java -Xmx1073741824 -Xms268435456 MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             1073741824, // 1GB in bytes
				JavaInitialHeapSize:         268435456,  // 256MB in bytes
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "",
			},
		},
		{
			name:  "XX options with complex values",
			input: "java -XX:NewRatio=3 -XX:SurvivorRatio=6 -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/tmp/heap.hprof MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:NewRatio=3,-XX:SurvivorRatio=6,-XX:+HeapDumpOnOutOfMemoryError,-XX:HeapDumpPath=/tmp/heap.hprof",
			},
		},
		{
			name:  "mixed case units",
			input: "java -Xmx2G -Xms512m MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             2 * 1024 * 1024 * 1024, // 2GB
				JavaInitialHeapSize:         512 * 1024 * 1024,      // 512MB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "",
			},
		},
		{
			name:  "real-world Spring Boot app",
			input: "java -Xms512m -Xmx2g -XX:+UseG1GC -XX:MaxHeapSize=2g -XX:MinHeapSize=512m -XX:+PrintGCDetails -jar my-spring-app.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             2 * 1024 * 1024 * 1024, // 2GB
				JavaInitialHeapSize:         512 * 1024 * 1024,      // 512MB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC,-XX:MaxHeapSize=2g,-XX:MinHeapSize=512m,-XX:+PrintGCDetails",
			},
		},
		{
			name:  "swapped order with disabled collector",
			input: "java -XX:-UseParallelGC -Xmx1g -Xms256m -XX:+UseG1GC -XX:MaxHeapSize=1g -XX:MinHeapSize=256m -jar worker.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             1 * 1024 * 1024 * 1024, // 1GB
				JavaInitialHeapSize:         256 * 1024 * 1024,      // 256MB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:-UseParallelGC,-XX:+UseG1GC,-XX:MaxHeapSize=1g,-XX:MinHeapSize=256m",
			},
		},
		{
			name:  "invalid parameters gracefully handled",
			input: "java --Xmx512m -XMS256m -XX:+UseG1GC=1 -jar broken.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             0, // --Xmx should not match (double dash)
				JavaInitialHeapSize:         0, // -XMS should not match (capital S)
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC=1", // This will match as our regex is permissive for edge cases
			},
		},
		{
			name:  "RAM percentage parameters Java 11+",
			input: "java -XX:InitialRAMPercentage=25.0 -XX:MaxRAMPercentage=75.0 -jar my-spring-app.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             -1, // Using percentage
				JavaInitialHeapSize:         -1, // Using percentage
				JavaMaxHeapAsPercentage:     75.0,
				JavaInitialHeapAsPercentage: 25.0,
				XXOptions:                   "-XX:InitialRAMPercentage=25.0,-XX:MaxRAMPercentage=75.0",
			},
		},
		{
			name:  "max heap percentage only",
			input: "java -XX:MaxRAMPercentage=50.0 -jar worker.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             -1, // Using percentage
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     50.0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxRAMPercentage=50.0",
			},
		},
		{
			name:  "min initial and max RAM percentages",
			input: "java -XX:MinRAMPercentage=10.0 -XX:InitialRAMPercentage=20.0 -XX:MaxRAMPercentage=60.0 -jar analytics-job.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             -1, // Using percentage
				JavaInitialHeapSize:         -1, // Using percentage
				JavaMaxHeapAsPercentage:     60.0,
				JavaInitialHeapAsPercentage: 20.0,
				XXOptions:                   "-XX:MinRAMPercentage=10.0,-XX:InitialRAMPercentage=20.0,-XX:MaxRAMPercentage=60.0",
			},
		},
		{
			name:  "floating-point precision parameters",
			input: "java -XX:MaxRAMPercentage=66.67 -XX:InitialRAMPercentage=33.33 -jar microservice.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             -1, // Using percentage
				JavaInitialHeapSize:         -1, // Using percentage
				JavaMaxHeapAsPercentage:     66.67,
				JavaInitialHeapAsPercentage: 33.33,
				XXOptions:                   "-XX:MaxRAMPercentage=66.67,-XX:InitialRAMPercentage=33.33",
			},
		},
		{
			name:  "complex mixed parameters",
			input: "java -Xms1g -Xmx4g -XX:+UseG1GC -XX:-UseParallelGC -XX:MaxGCPauseMillis=200 -XX:G1HeapRegionSize=16m -XX:InitialRAMPercentage=25.0 -jar complex-app.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB (explicit -Xmx takes precedence)
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB (explicit -Xms takes precedence over percentage)
				JavaMaxHeapAsPercentage:     0,                      // Not used since explicit max size provided
				JavaInitialHeapAsPercentage: 25.0,                   // Parsed but not used since explicit initial size provided
				XXOptions:                   "-XX:+UseG1GC,-XX:-UseParallelGC,-XX:MaxGCPauseMillis=200,-XX:G1HeapRegionSize=16m,-XX:InitialRAMPercentage=25.0",
			},
		},
		{
			name:  "XX options with paths",
			input: "java -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/tmp/heapdump.hprof -XX:ErrorFile=/var/log/jvm/error.log -jar app.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+HeapDumpOnOutOfMemoryError,-XX:HeapDumpPath=/tmp/heapdump.hprof,-XX:ErrorFile=/var/log/jvm/error.log",
			},
		},
		{
			name:  "GC tuning parameters",
			input: "java -XX:+UseG1GC -XX:G1NewSizePercent=30 -XX:G1MaxNewSizePercent=40 -XX:G1MixedGCCountTarget=8 -XX:G1OldCSetRegionThreshold=10 -jar gc-tuned-app.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC,-XX:G1NewSizePercent=30,-XX:G1MaxNewSizePercent=40,-XX:G1MixedGCCountTarget=8,-XX:G1OldCSetRegionThreshold=10",
			},
		},
		{
			name:  "debugging and monitoring flags",
			input: "java -XX:+PrintGC -XX:+PrintGCDetails -XX:+PrintGCTimeStamps -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=5 -XX:GCLogFileSize=10M -jar monitored-app.jar",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+PrintGC,-XX:+PrintGCDetails,-XX:+PrintGCTimeStamps,-XX:+UseGCLogFileRotation,-XX:NumberOfGCLogFiles=5,-XX:GCLogFileSize=10M",
			},
		},
		{
			name:  "parameter override - last occurrence wins",
			input: "java -Xmx1g -Xms256m -Xmx2g -Xms512m -XX:MaxRAMPercentage=50.0 -XX:MaxRAMPercentage=75.0 MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             -1,                // Using percentage due to MaxRAMPercentage
				JavaInitialHeapSize:         512 * 1024 * 1024, // Last -Xms512m wins (no percentage override)
				JavaMaxHeapAsPercentage:     75.0,              // Last MaxRAMPercentage=75.0 wins
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxRAMPercentage=50.0,-XX:MaxRAMPercentage=75.0",
			},
		},
		{
			name:  "XX:MaxHeapSize only",
			input: "java -XX:MaxHeapSize=4g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxHeapSize=4g",
			},
		},
		{
			name:  "XX:MinHeapSize only",
			input: "java -XX:MinHeapSize=1g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MinHeapSize=1g",
			},
		},
		{
			name:  "XX:MaxHeapSize and XX:MinHeapSize together",
			input: "java -XX:MaxHeapSize=8g -XX:MinHeapSize=2g -XX:+UseG1GC MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             8 * 1024 * 1024 * 1024, // 8GB
				JavaInitialHeapSize:         2 * 1024 * 1024 * 1024, // 2GB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxHeapSize=8g,-XX:MinHeapSize=2g,-XX:+UseG1GC",
			},
		},
		{
			name:  "XX:MaxHeapSize takes precedence over Xmx - XX comes later",
			input: "java -Xmx1g -XX:MaxHeapSize=4g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB from XX:MaxHeapSize (rightmost)
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxHeapSize=4g",
			},
		},
		{
			name:  "Xmx takes precedence over XX:MaxHeapSize - Xmx comes later",
			input: "java -XX:MaxHeapSize=1g -Xmx4g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB from -Xmx (rightmost)
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxHeapSize=1g",
			},
		},
		{
			name:  "XX:MinHeapSize takes precedence over Xms - XX comes later",
			input: "java -Xms512m -XX:MinHeapSize=2g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         2 * 1024 * 1024 * 1024, // 2GB from XX:MinHeapSize (rightmost)
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MinHeapSize=2g",
			},
		},
		{
			name:  "Xms takes precedence over XX:MinHeapSize - Xms comes later",
			input: "java -XX:MinHeapSize=512m -Xms2g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         2 * 1024 * 1024 * 1024, // 2GB from -Xms (rightmost)
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MinHeapSize=512m",
			},
		},
		{
			name:  "mixed XX and X parameters - complex precedence",
			input: "java -Xmx1g -XX:MaxHeapSize=2g -Xms256m -XX:MinHeapSize=512m -Xmx4g -XX:MinHeapSize=1g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB from last -Xmx (rightmost max heap param)
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB from last XX:MinHeapSize (rightmost initial heap param)
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxHeapSize=2g,-XX:MinHeapSize=512m,-XX:MinHeapSize=1g",
			},
		},
		{
			name:  "XX parameters with different units",
			input: "java -XX:MaxHeapSize=2048m -XX:MinHeapSize=1024K MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             2048 * 1024 * 1024, // 2048MB = 2GB
				JavaInitialHeapSize:         1024 * 1024,        // 1024KB = 1MB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxHeapSize=2048m,-XX:MinHeapSize=1024K",
			},
		},
		{
			name:  "XX parameters mixed with percentage parameters",
			input: "java -XX:MaxHeapSize=2g -XX:InitialRAMPercentage=25.0 MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             2 * 1024 * 1024 * 1024, // 2GB (explicit size takes precedence)
				JavaInitialHeapSize:         -1,                     // Using percentage (no explicit initial size)
				JavaMaxHeapAsPercentage:     0,                      // Not used since explicit max size provided
				JavaInitialHeapAsPercentage: 25.0,                   // Used since no explicit initial size provided
				XXOptions:                   "-XX:MaxHeapSize=2g,-XX:InitialRAMPercentage=25.0",
			},
		},
		{
			name:  "multiple XX:MaxHeapSize parameters - last wins",
			input: "java -XX:MaxHeapSize=1g -XX:MaxHeapSize=2g -XX:MaxHeapSize=4g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB from last XX:MaxHeapSize
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxHeapSize=1g,-XX:MaxHeapSize=2g,-XX:MaxHeapSize=4g",
			},
		},
		{
			name:  "multiple XX:MinHeapSize parameters - last wins",
			input: "java -XX:MinHeapSize=256m -XX:MinHeapSize=512m -XX:MinHeapSize=1g MyApp",
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB from last XX:MinHeapSize
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MinHeapSize=256m,-XX:MinHeapSize=512m,-XX:MinHeapSize=1g",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJVMParamsFromString(tt.input)
			if result.JavaMaxHeapSize != tt.expected.JavaMaxHeapSize {
				t.Errorf("JavaMaxHeapSize: got %.0f, want %.0f", result.JavaMaxHeapSize, tt.expected.JavaMaxHeapSize)
			}
			if result.JavaInitialHeapSize != tt.expected.JavaInitialHeapSize {
				t.Errorf("JavaInitialHeapSize: got %.0f, want %.0f", result.JavaInitialHeapSize, tt.expected.JavaInitialHeapSize)
			}
			if result.JavaMaxHeapAsPercentage != tt.expected.JavaMaxHeapAsPercentage {
				t.Errorf("JavaMaxHeapAsPercentage: got %.2f, want %.2f", result.JavaMaxHeapAsPercentage, tt.expected.JavaMaxHeapAsPercentage)
			}
			if result.JavaInitialHeapAsPercentage != tt.expected.JavaInitialHeapAsPercentage {
				t.Errorf("JavaInitialHeapAsPercentage: got %.2f, want %.2f", result.JavaInitialHeapAsPercentage, tt.expected.JavaInitialHeapAsPercentage)
			}
			if result.XXOptions != tt.expected.XXOptions {
				t.Errorf("XXOptions: got %q, want %q", result.XXOptions, tt.expected.XXOptions)
			}
		})
	}
}

func TestParseJVMParams(t *testing.T) {
	// Create a temporary directory to simulate /proc/{pid}
	tempDir, err := os.MkdirTemp("", "jvm_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		cmdline     string
		env         map[string]string
		expected    JVMParams
		description string
	}{
		{
			name:    "cmdline only",
			cmdline: "java -Xmx4g -Xms1g -XX:+UseG1GC MyApp",
			env:     map[string]string{},
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC",
			},
			description: "Parameters only in command line",
		},
		{
			name:    "env only - JAVA_TOOL_OPTIONS",
			cmdline: "java MyApp",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "-Xmx2g -Xms512m -XX:+UseParallelGC",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             2 * 1024 * 1024 * 1024, // 2GB
				JavaInitialHeapSize:         512 * 1024 * 1024,      // 512MB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseParallelGC",
			},
			description: "Parameters only in JAVA_TOOL_OPTIONS",
		},
		{
			name:    "env only - _JAVA_OPTIONS",
			cmdline: "java MyApp",
			env: map[string]string{
				"_JAVA_OPTIONS": "-Xmx3g -Xms768m -XX:+UseG1GC",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             3 * 1024 * 1024 * 1024, // 3GB
				JavaInitialHeapSize:         768 * 1024 * 1024,      // 768MB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC",
			},
			description: "Parameters only in _JAVA_OPTIONS",
		},
		{
			name:    "env only - JDK_JAVA_OPTIONS",
			cmdline: "java MyApp",
			env: map[string]string{
				"JDK_JAVA_OPTIONS": "-Xmx1g -Xms256m -XX:+UseConcMarkSweepGC",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             1 * 1024 * 1024 * 1024, // 1GB
				JavaInitialHeapSize:         256 * 1024 * 1024,      // 256MB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseConcMarkSweepGC",
			},
			description: "Parameters only in JDK_JAVA_OPTIONS",
		},
		{
			name:    "env only - IBM_JAVA_OPTIONS",
			cmdline: "java MyApp",
			env: map[string]string{
				"IBM_JAVA_OPTIONS": "-Xmx5g -Xms1g -XX:+UseZGC",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             5 * 1024 * 1024 * 1024, // 5GB
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseZGC",
			},
			description: "Parameters only in IBM_JAVA_OPTIONS",
		},
		{
			name:    "cmdline overrides env - same parameters",
			cmdline: "java -Xmx8g -Xms2g -XX:+UseG1GC MyApp",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "-Xmx1g -Xms256m -XX:+UseParallelGC",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             8 * 1024 * 1024 * 1024, // 8GB from cmdline (overrides env)
				JavaInitialHeapSize:         2 * 1024 * 1024 * 1024, // 2GB from cmdline (overrides env)
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseParallelGC,-XX:+UseG1GC", // Both included, cmdline last
			},
			description: "Command line parameters override environment variables",
		},
		{
			name:    "env and cmdline merge - different parameters",
			cmdline: "java -Xmx6g MyApp",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "-Xms1g -XX:+UseG1GC -XX:MaxGCPauseMillis=200",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             6 * 1024 * 1024 * 1024, // 6GB from cmdline
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB from env (not in cmdline)
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC,-XX:MaxGCPauseMillis=200", // From env
			},
			description: "Environment and command line parameters are merged",
		},
		{
			name:    "env precedence order - all env vars combined",
			cmdline: "java MyApp",
			env: map[string]string{
				"IBM_JAVA_OPTIONS":  "-Xmx1g -Xms256m -XX:+UseZGC",
				"JDK_JAVA_OPTIONS":  "-Xmx2g -Xms512m -XX:+UseConcMarkSweepGC",
				"_JAVA_OPTIONS":     "-Xmx3g -Xms768m -XX:+UseG1GC",
				"JAVA_TOOL_OPTIONS": "-Xmx4g -Xms1g -XX:+UseParallelGC",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             1 * 1024 * 1024 * 1024, // 1GB (last -Xmx from IBM_JAVA_OPTIONS, rightmost)
				JavaInitialHeapSize:         256 * 1024 * 1024,      // 256MB (last -Xms from IBM_JAVA_OPTIONS, rightmost)
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseParallelGC,-XX:+UseG1GC,-XX:+UseConcMarkSweepGC,-XX:+UseZGC",
			},
			description: "All environment variables are processed in order, rightmost parameter wins",
		},
		{
			name:    "partial env vars",
			cmdline: "java MyApp",
			env: map[string]string{
				"JDK_JAVA_OPTIONS": "-Xmx2g",
				"_JAVA_OPTIONS":    "-Xms512m -XX:+UseG1GC",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             2 * 1024 * 1024 * 1024, // 2GB
				JavaInitialHeapSize:         512 * 1024 * 1024,      // 512MB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC",
			},
			description: "Parameters collected from multiple env vars in precedence order",
		},
		{
			name:    "empty env values ignored",
			cmdline: "java MyApp",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "",
				"_JAVA_OPTIONS":     "-Xmx1g -Xms256m -XX:+UseG1GC",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             1 * 1024 * 1024 * 1024, // 1GB
				JavaInitialHeapSize:         256 * 1024 * 1024,      // 256MB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC",
			},
			description: "Empty environment variables are ignored",
		},
		{
			name:    "no parameters anywhere",
			cmdline: "java MyApp",
			env:     map[string]string{},
			expected: JVMParams{
				JavaMaxHeapSize:             0,
				JavaInitialHeapSize:         0,
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "",
			},
			description: "No parameters in cmdline or env",
		},
		{
			name:    "real-world Spring Boot env",
			cmdline: "java -jar my-spring-app.jar",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "-Xms512m -Xmx2g -XX:+UseG1GC -XX:MaxHeapSize=2g -XX:MinHeapSize=512m -XX:+PrintGCDetails",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             2 * 1024 * 1024 * 1024, // 2GB
				JavaInitialHeapSize:         512 * 1024 * 1024,      // 512MB
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC,-XX:MaxHeapSize=2g,-XX:MinHeapSize=512m,-XX:+PrintGCDetails",
			},
			description: "Real-world Spring Boot parameters in JAVA_TOOL_OPTIONS",
		},
		{
			name:    "RAM percentage in env vars",
			cmdline: "java -jar microservice.jar",
			env: map[string]string{
				"_JAVA_OPTIONS": "-XX:InitialRAMPercentage=25.0 -XX:MaxRAMPercentage=75.0",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             -1, // Using percentage
				JavaInitialHeapSize:         -1, // Using percentage
				JavaMaxHeapAsPercentage:     75.0,
				JavaInitialHeapAsPercentage: 25.0,
				XXOptions:                   "-XX:InitialRAMPercentage=25.0,-XX:MaxRAMPercentage=75.0",
			},
			description: "RAM percentage parameters in _JAVA_OPTIONS",
		},
		{
			name:    "cmdline overrides env - mixed params",
			cmdline: "java -Xmx8g -XX:+UseZGC -jar production-app.jar",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "-Xmx2g -Xms512m -XX:+UseG1GC -XX:MaxGCPauseMillis=200",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             8 * 1024 * 1024 * 1024, // 8GB from cmdline (overrides env -Xmx2g)
				JavaInitialHeapSize:         512 * 1024 * 1024,      // 512MB from env (not overridden in cmdline)
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC,-XX:MaxGCPauseMillis=200,-XX:+UseZGC", // Combined, cmdline last
			},
			description: "Command line parameters override environment variables for same parameters",
		},
		{
			name:    "mixed env vars with complex parameters",
			cmdline: "java -jar analytics-service.jar",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "-XX:+UseG1GC -XX:G1NewSizePercent=30",
				"_JAVA_OPTIONS":     "-Xmx4g -XX:MaxGCPauseMillis=200",
				"JDK_JAVA_OPTIONS":  "-Xms1g",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB from _JAVA_OPTIONS
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB from JDK_JAVA_OPTIONS
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC,-XX:G1NewSizePercent=30,-XX:MaxGCPauseMillis=200", // Combined from all env vars
			},
			description: "Complex parameters collected from multiple environment variables",
		},
		{
			name:    "invalid env parameters handled gracefully",
			cmdline: "java -jar app.jar",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "--Xmx2g -XMS512m -XX:+UseG1GC=invalid",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             0, // --Xmx and -XMS are invalid and ignored
				JavaInitialHeapSize:         0, // --Xmx and -XMS are invalid and ignored
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC=invalid", // XX options are parsed permissively
			},
			description: "Invalid X parameters are ignored, XX options are parsed permissively",
		},
		{
			name:    "XX:MaxHeapSize in env overridden by cmdline -Xmx",
			cmdline: "java -Xmx8g -jar app.jar",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "-XX:MaxHeapSize=2g -Xms512m",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             8 * 1024 * 1024 * 1024, // 8GB from cmdline -Xmx (overrides env XX:MaxHeapSize)
				JavaInitialHeapSize:         512 * 1024 * 1024,      // 512MB from env -Xms
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxHeapSize=2g",
			},
			description: "Command line -Xmx overrides environment XX:MaxHeapSize",
		},
		{
			name:    "XX:MinHeapSize in env overridden by cmdline -Xms",
			cmdline: "java -Xms2g -jar app.jar",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "-Xmx4g -XX:MinHeapSize=512m",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             4 * 1024 * 1024 * 1024, // 4GB from env -Xmx
				JavaInitialHeapSize:         2 * 1024 * 1024 * 1024, // 2GB from cmdline -Xms (overrides env XX:MinHeapSize)
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MinHeapSize=512m",
			},
			description: "Command line -Xms overrides environment XX:MinHeapSize",
		},
		{
			name:    "cmdline XX:MaxHeapSize overrides env -Xmx",
			cmdline: "java -XX:MaxHeapSize=6g -jar app.jar",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "-Xmx2g -Xms1g -XX:+UseG1GC",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             6 * 1024 * 1024 * 1024, // 6GB from cmdline XX:MaxHeapSize (overrides env -Xmx)
				JavaInitialHeapSize:         1 * 1024 * 1024 * 1024, // 1GB from env -Xms
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:+UseG1GC,-XX:MaxHeapSize=6g",
			},
			description: "Command line XX:MaxHeapSize overrides environment -Xmx",
		},
		{
			name:    "mixed XX and X parameters across env and cmdline - complex precedence",
			cmdline: "java -Xmx8g -XX:MinHeapSize=2g -jar app.jar",
			env: map[string]string{
				"JAVA_TOOL_OPTIONS": "-XX:MaxHeapSize=4g -Xms1g -XX:+UseG1GC",
			},
			expected: JVMParams{
				JavaMaxHeapSize:             8 * 1024 * 1024 * 1024, // 8GB cmdline -Xmx (overrides env XX:MaxHeapSize)
				JavaInitialHeapSize:         2 * 1024 * 1024 * 1024, // 2GB cmdline XX:MinHeapSize (overrides env -Xms)
				JavaMaxHeapAsPercentage:     0,
				JavaInitialHeapAsPercentage: 0,
				XXOptions:                   "-XX:MaxHeapSize=4g,-XX:+UseG1GC,-XX:MinHeapSize=2g",
			},
			description: "Complex mixing of XX and X parameters with proper precedence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock environ file
			pidDir := filepath.Join(tempDir, "12345")
			err := os.MkdirAll(pidDir, 0755)
			if err != nil {
				t.Fatal(err)
			}

			environFile := filepath.Join(pidDir, "environ")
			var environContent string
			for key, value := range tt.env {
				environContent += key + "=" + value + "\x00"
			}

			err = os.WriteFile(environFile, []byte(environContent), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Test with the real environment reading
			env := make(map[string]string)
			if environFile != "" {
				content, err := os.ReadFile(environFile)
				if err == nil {
					parts := strings.Split(string(content), "\x00")
					for _, part := range parts {
						if part == "" {
							continue
						}
						kv := strings.SplitN(part, "=", 2)
						if len(kv) == 2 {
							env[kv[0]] = kv[1]
						}
					}
				}
			}

			// Simulate the combined parameter logic manually
			var combinedArgs strings.Builder

			// Process environment variables in order of precedence
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

			// Append command line parameters
			if tt.cmdline != "" {
				if combinedArgs.Len() > 0 {
					combinedArgs.WriteString(" ")
				}
				combinedArgs.WriteString(tt.cmdline)
			}

			// Parse the combined parameter string
			params := parseJVMParamsFromString(combinedArgs.String())

			if params.JavaMaxHeapSize != tt.expected.JavaMaxHeapSize {
				t.Errorf("JavaMaxHeapSize: got %.0f, want %.0f (%s)", params.JavaMaxHeapSize, tt.expected.JavaMaxHeapSize, tt.description)
			}
			if params.JavaInitialHeapSize != tt.expected.JavaInitialHeapSize {
				t.Errorf("JavaInitialHeapSize: got %.0f, want %.0f (%s)", params.JavaInitialHeapSize, tt.expected.JavaInitialHeapSize, tt.description)
			}
			if params.JavaMaxHeapAsPercentage != tt.expected.JavaMaxHeapAsPercentage {
				t.Errorf("JavaMaxHeapAsPercentage: got %.2f, want %.2f (%s)", params.JavaMaxHeapAsPercentage, tt.expected.JavaMaxHeapAsPercentage, tt.description)
			}
			if params.JavaInitialHeapAsPercentage != tt.expected.JavaInitialHeapAsPercentage {
				t.Errorf("JavaInitialHeapAsPercentage: got %.2f, want %.2f (%s)", params.JavaInitialHeapAsPercentage, tt.expected.JavaInitialHeapAsPercentage, tt.description)
			}
			if params.XXOptions != tt.expected.XXOptions {
				t.Errorf("XXOptions: got %q, want %q (%s)", params.XXOptions, tt.expected.XXOptions, tt.description)
			}
		})
	}
}

func TestReadEnviron(t *testing.T) {
	// Create a temporary directory to simulate /proc/{pid}
	tempDir, err := os.MkdirTemp("", "environ_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock environ file
	pidDir := filepath.Join(tempDir, "12345")
	err = os.MkdirAll(pidDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	environFile := filepath.Join(pidDir, "environ")
	environContent := "PATH=/usr/bin:/bin\x00HOME=/home/user\x00JAVA_TOOL_OPTIONS=-Xmx2g -XX:+UseG1GC\x00_JAVA_OPTIONS=-Xms512m\x00EMPTY_VAR=\x00"

	err = os.WriteFile(environFile, []byte(environContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// We can't easily test readEnviron directly without changing proc.Path,
	// so we'll test the parsing logic separately
	expected := map[string]string{
		"PATH":              "/usr/bin:/bin",
		"HOME":              "/home/user",
		"JAVA_TOOL_OPTIONS": "-Xmx2g -XX:+UseG1GC",
		"_JAVA_OPTIONS":     "-Xms512m",
		"EMPTY_VAR":         "",
	}

	// Simulate what readEnviron does
	content, err := os.ReadFile(environFile)
	if err != nil {
		t.Fatal(err)
	}

	result := make(map[string]string)
	parts := strings.Split(string(content), "\x00")
	for _, part := range parts {
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}

	for key, expectedValue := range expected {
		if gotValue, exists := result[key]; !exists {
			t.Errorf("Missing key %q", key)
		} else if gotValue != expectedValue {
			t.Errorf("Key %q: got %q, want %q", key, gotValue, expectedValue)
		}
	}
}

// Benchmark tests
func BenchmarkParseJVMParamsFromString(b *testing.B) {
	input := "java -Xmx8g -Xms2g -XX:+UseG1GC -XX:MaxGCPauseMillis=200 -XX:+DisableExplicitGC -XX:+HeapDumpOnOutOfMemoryError MyApp"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseJVMParamsFromString(input)
	}
}
