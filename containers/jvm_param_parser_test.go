package containers

import (
	"testing"
)

func TestDetectJVMVendor(t *testing.T) {
	tests := []struct {
		name     string
		vmFlags  string
		expected JVMVendor
	}{
		{
			name:     "HotSpot detection from flags",
			vmFlags:  "-XX:MaxHeapSize=2147483648 -XX:+UseG1GC HotSpot VM",
			expected: JVMVendorHotSpot,
		},
		{
			name:     "OpenJ9 detection from flags",
			vmFlags:  "-Xmx2g -Xms512m OpenJ9 VM",
			expected: JVMVendorOpenJ9,
		},
		{
			name:     "GraalVM detection from flags",
			vmFlags:  "-XX:MaxHeapSize=1073741824 GraalVM Enterprise",
			expected: JVMVendorGraalVM,
		},
		{
			name:     "IBM J9 detection from flags",
			vmFlags:  "-Xgcpolicy:gencon IBM J9 VM",
			expected: JVMVendorOpenJ9,
		},
		{
			name:     "Unknown vendor",
			vmFlags:  "some unknown VM flags",
			expected: JVMVendorUnknown,
		},
		{
			name:     "HotSpot style flags fallback",
			vmFlags:  "-XX:MaxHeapSize=2147483648 -XX:InitialHeapSize=268435456",
			expected: JVMVendorHotSpot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectJVMVendor(tt.vmFlags)
			if result != tt.expected {
				t.Errorf("detectJVMVendor() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseHotSpotVMFlags(t *testing.T) {
	tests := []struct {
		name     string
		vmFlags  string
		expected JVMParams
	}{
		{
			name: "Standard HotSpot flags",
			vmFlags: "-XX:MaxHeapSize=2147483648 -XX:InitialHeapSize=268435456 " +
				"-XX:MaxRAMPercentage=75.0 -XX:InitialRAMPercentage=25.0",
			expected: JVMParams{
				JavaMaxHeapSize:             "2147483648",
				JavaInitialHeapSize:         "268435456",
				JavaMaxHeapAsPercentage:     "75.0",
				JavaInitialHeapAsPercentage: "25.0",
			},
		},
		{
			name:    "Empty flags",
			vmFlags: "",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "",
				JavaMaxHeapAsPercentage:     "",
				JavaInitialHeapAsPercentage: "",
			},
		},
		{
			name:    "MinHeapSize flag (legacy)",
			vmFlags: "-XX:MinHeapSize=134217728",
			expected: JVMParams{
				JavaMaxHeapSize:             "",
				JavaInitialHeapSize:         "134217728",
				JavaMaxHeapAsPercentage:     "",
				JavaInitialHeapAsPercentage: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHotSpotVMFlags(tt.vmFlags)
			if result.JavaMaxHeapSize != tt.expected.JavaMaxHeapSize ||
				result.JavaInitialHeapSize != tt.expected.JavaInitialHeapSize ||
				result.JavaMaxHeapAsPercentage != tt.expected.JavaMaxHeapAsPercentage ||
				result.JavaInitialHeapAsPercentage != tt.expected.JavaInitialHeapAsPercentage {
				t.Errorf("parseHotSpotVMFlags() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestParseOpenJ9VMFlags(t *testing.T) {
	tests := []struct {
		name     string
		vmFlags  string
		expected JVMParams
	}{
		{
			name:    "OpenJ9 Xmx/Xms flags",
			vmFlags: "-Xmx2g -Xms512m",
			expected: JVMParams{
				JavaMaxHeapSize:     "2g",
				JavaInitialHeapSize: "512m",
			},
		},
		{
			name:    "Mixed OpenJ9 and HotSpot flags",
			vmFlags: "-Xmx1g -XX:MaxHeapSize=1073741824",
			expected: JVMParams{
				JavaMaxHeapSize: "1073741824", // HotSpot flag should take precedence
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOpenJ9VMFlags(tt.vmFlags)
			if result.JavaMaxHeapSize != tt.expected.JavaMaxHeapSize ||
				result.JavaInitialHeapSize != tt.expected.JavaInitialHeapSize {
				t.Errorf("parseOpenJ9VMFlags() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestParseHotSpotGCType(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		expected string
	}{
		{
			name:     "G1GC detection",
			flags:    []string{"-XX:+UseG1GC", "-XX:MaxGCPauseMillis=200"},
			expected: "G1GC",
		},
		{
			name:     "ZGC detection",
			flags:    []string{"-XX:+UseZGC", "-XX:+UnlockExperimentalVMOptions"},
			expected: "ZGC",
		},
		{
			name:     "Parallel GC detection",
			flags:    []string{"-XX:+UseParallelGC"},
			expected: "ParallelGC",
		},
		{
			name:     "Serial GC detection",
			flags:    []string{"-XX:+UseSerialGC"},
			expected: "SerialGC",
		},
		{
			name:     "CMS GC detection",
			flags:    []string{"-XX:+UseConcMarkSweepGC"},
			expected: "ConcMarkSweepGC",
		},
		{
			name:     "Shenandoah GC detection",
			flags:    []string{"-XX:+UseShenandoahGC"},
			expected: "ShenandoahGC",
		},
		{
			name:     "No explicit GC flags",
			flags:    []string{"-XX:MaxHeapSize=2147483648"},
			expected: "Unknown",
		},
		{
			name:     "G1 inference from other flags",
			flags:    []string{"-XX:G1HeapRegionSize=16m"},
			expected: "G1GC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHotSpotGCType(tt.flags)
			if result != tt.expected {
				t.Errorf("parseHotSpotGCType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseOpenJ9GCType(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		expected string
	}{
		{
			name:     "Gencon GC policy",
			flags:    []string{"-Xgcpolicy:gencon"},
			expected: "Generational Concurrent (OpenJ9)",
		},
		{
			name:     "Throughput GC policy",
			flags:    []string{"-Xgcpolicy:optthruput"},
			expected: "Throughput (OpenJ9)",
		},
		{
			name:     "Average pause GC policy",
			flags:    []string{"-Xgcpolicy:optavgpause"},
			expected: "Average Pause (OpenJ9)",
		},
		{
			name:     "Balanced GC policy",
			flags:    []string{"-Xgcpolicy:balanced"},
			expected: "Balanced (OpenJ9)",
		},
		{
			name:     "No GC policy specified",
			flags:    []string{"-Xmx2g", "-Xms512m"},
			expected: "Unknown (OpenJ9)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOpenJ9GCType(tt.flags)
			if result != tt.expected {
				t.Errorf("parseOpenJ9GCType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseFromCmdline(t *testing.T) {
	tests := []struct {
		name     string
		cmdline  string
		expected JVMParams
	}{
		{
			name:    "Standard Java command line",
			cmdline: "java -Xmx2g -Xms512m -XX:+UseG1GC -jar app.jar",
			expected: JVMParams{
				JavaMaxHeapSize:     "2g",
				JavaInitialHeapSize: "512m",
				GCType:              "G1GC",
			},
		},
		{
			name:    "HotSpot style flags in command line",
			cmdline: "java -XX:MaxHeapSize=2147483648 -XX:InitialHeapSize=268435456 -XX:+UseParallelGC",
			expected: JVMParams{
				JavaMaxHeapSize:     "2147483648",
				JavaInitialHeapSize: "268435456",
				GCType:              "ParallelGC",
			},
		},
		{
			name:    "OpenJ9 GC policy",
			cmdline: "java -Xmx1g -Xgcpolicy:balanced -jar app.jar",
			expected: JVMParams{
				JavaMaxHeapSize: "1g",
				GCType:          "Balanced (OpenJ9)",
			},
		},
		{
			name:    "Percentage-based heap sizing",
			cmdline: "java -XX:MaxRAMPercentage=75.0 -XX:InitialRAMPercentage=25.0 -jar app.jar",
			expected: JVMParams{
				JavaMaxHeapAsPercentage:     "75.0",
				JavaInitialHeapAsPercentage: "25.0",
				GCType:                      "Unknown",
			},
		},
		{
			name:    "No heap flags",
			cmdline: "java -jar app.jar",
			expected: JVMParams{
				GCType: "Unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFromCmdline(tt.cmdline, 12345)
			if result.JavaMaxHeapSize != tt.expected.JavaMaxHeapSize ||
				result.JavaInitialHeapSize != tt.expected.JavaInitialHeapSize ||
				result.JavaMaxHeapAsPercentage != tt.expected.JavaMaxHeapAsPercentage ||
				result.JavaInitialHeapAsPercentage != tt.expected.JavaInitialHeapAsPercentage ||
				result.GCType != tt.expected.GCType {
				t.Errorf("parseFromCmdline() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name     string
		sizeStr  string
		expected string
	}{
		{
			name:     "Numeric bytes",
			sizeStr:  "2147483648",
			expected: "2147483648",
		},
		{
			name:     "Kilobytes",
			sizeStr:  "512k",
			expected: "512k",
		},
		{
			name:     "Megabytes",
			sizeStr:  "2048m",
			expected: "2048m",
		},
		{
			name:     "Gigabytes",
			sizeStr:  "4g",
			expected: "4g",
		},
		{
			name:     "Terabytes",
			sizeStr:  "1t",
			expected: "1t",
		},
		{
			name:     "Empty string",
			sizeStr:  "",
			expected: "",
		},
		{
			name:     "Invalid format",
			sizeStr:  "invalid",
			expected: "",
		},
		{
			name:     "Case insensitive",
			sizeStr:  "2G",
			expected: "2g",
		},
		{
			name:     "With 'b' suffix",
			sizeStr:  "1024kb",
			expected: "1024k",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSize(tt.sizeStr)
			if result != tt.expected {
				t.Errorf("parseSize() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestJVMVendorString(t *testing.T) {
	tests := []struct {
		vendor   JVMVendor
		expected string
	}{
		{JVMVendorHotSpot, "HotSpot"},
		{JVMVendorOpenJ9, "OpenJ9"},
		{JVMVendorGraalVM, "GraalVM"},
		{JVMVendorUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.vendor.String()
			if result != tt.expected {
				t.Errorf("JVMVendor.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseGCTypeFromCmdline(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "ZGC from command line",
			args:     []string{"java", "-XX:+UseZGC", "-jar", "app.jar"},
			expected: "ZGC",
		},
		{
			name:     "OpenJ9 gencon policy",
			args:     []string{"java", "-Xgcpolicy:gencon", "-jar", "app.jar"},
			expected: "Generational Concurrent (OpenJ9)",
		},
		{
			name:     "No GC flags",
			args:     []string{"java", "-Xmx2g", "-jar", "app.jar"},
			expected: "Unknown",
		},
		{
			name:     "Multiple GC flags - last one wins",
			args:     []string{"java", "-XX:+UseParallelGC", "-XX:+UseG1GC", "-jar", "app.jar"},
			expected: "G1GC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGCTypeFromCmdline(tt.args)
			if result != tt.expected {
				t.Errorf("parseGCTypeFromCmdline() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkDetectJVMVendor(b *testing.B) {
	vmFlags := "-XX:MaxHeapSize=2147483648 -XX:InitialHeapSize=268435456 -XX:+UseG1GC HotSpot VM"

	for i := 0; i < b.N; i++ {
		detectJVMVendor(vmFlags)
	}
}

func BenchmarkParseHotSpotVMFlags(b *testing.B) {
	vmFlags := "-XX:MaxHeapSize=2147483648 -XX:InitialHeapSize=268435456 " +
		"-XX:MaxRAMPercentage=75.0 -XX:InitialRAMPercentage=25.0 -XX:+UseG1GC"

	for i := 0; i < b.N; i++ {
		parseHotSpotVMFlags(vmFlags)
	}
}

func BenchmarkParseFromCmdline(b *testing.B) {
	cmdline := "java -Xmx2g -Xms512m -XX:+UseG1GC -XX:MaxRAMPercentage=75.0 -jar app.jar"

	for i := 0; i < b.N; i++ {
		parseFromCmdline(cmdline, 12345)
	}
}
