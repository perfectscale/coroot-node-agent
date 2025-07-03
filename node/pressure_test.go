package node

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPressureStats(t *testing.T) {
	// Create a temporary directory with test pressure files
	tmpDir := t.TempDir()
	pressureDir := path.Join(tmpDir, "pressure")
	require.NoError(t, os.MkdirAll(pressureDir, 0755))

	// Create test memory pressure file
	memoryContent := `some avg10=12.34 avg60=5.67 avg300=1.23 total=12345678
full avg10=0.50 avg60=0.25 avg300=0.10 total=987654
`
	require.NoError(t, os.WriteFile(path.Join(pressureDir, "memory"), []byte(memoryContent), 0644))

	// Create test CPU pressure file
	cpuContent := `some avg10=25.00 avg60=15.00 avg300=8.50 total=23456789
full avg10=0.00 avg60=0.00 avg300=0.00 total=0
`
	require.NoError(t, os.WriteFile(path.Join(pressureDir, "cpu"), []byte(cpuContent), 0644))

	// Create test IO pressure file
	ioContent := `some avg10=3.45 avg60=2.10 avg300=1.80 total=34567890
full avg10=1.20 avg60=0.80 avg300=0.60 total=12345678
`
	require.NoError(t, os.WriteFile(path.Join(pressureDir, "io"), []byte(ioContent), 0644))

	// Test GetSystemPressure
	pressure, err := GetSystemPressure(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, pressure)

	// Verify memory pressure
	assert.Equal(t, 12.34, pressure.Memory.Some.Avg10)
	assert.Equal(t, 5.67, pressure.Memory.Some.Avg60)
	assert.Equal(t, 1.23, pressure.Memory.Some.Avg300)
	assert.Equal(t, uint64(12345678), pressure.Memory.Some.Total)
	assert.Equal(t, 0.50, pressure.Memory.Full.Avg10)
	assert.Equal(t, 0.25, pressure.Memory.Full.Avg60)
	assert.Equal(t, 0.10, pressure.Memory.Full.Avg300)
	assert.Equal(t, uint64(987654), pressure.Memory.Full.Total)

	// Verify CPU pressure
	assert.Equal(t, 25.00, pressure.CPU.Some.Avg10)
	assert.Equal(t, 15.00, pressure.CPU.Some.Avg60)
	assert.Equal(t, 8.50, pressure.CPU.Some.Avg300)
	assert.Equal(t, uint64(23456789), pressure.CPU.Some.Total)

	// Verify IO pressure
	assert.Equal(t, 3.45, pressure.IO.Some.Avg10)
	assert.Equal(t, 2.10, pressure.IO.Some.Avg60)
	assert.Equal(t, 1.80, pressure.IO.Some.Avg300)
	assert.Equal(t, uint64(34567890), pressure.IO.Some.Total)
	assert.Equal(t, 1.20, pressure.IO.Full.Avg10)
	assert.Equal(t, 0.80, pressure.IO.Full.Avg60)
	assert.Equal(t, 0.60, pressure.IO.Full.Avg300)
	assert.Equal(t, uint64(12345678), pressure.IO.Full.Total)
}

func TestMemoryPressureLevels(t *testing.T) {
	tests := []struct {
		name     string
		some10   float64
		some60   float64
		some300  float64
		full10   float64
		expected string
	}{
		{
			name:     "no pressure",
			some10:   0.0,
			some60:   0.0,
			some300:  0.0,
			full10:   0.0,
			expected: "none",
		},
		{
			name:     "low pressure",
			some10:   5.0,
			some60:   2.0,
			some300:  0.5,
			full10:   0.0,
			expected: "low",
		},
		{
			name:     "medium pressure",
			some10:   15.0,
			some60:   8.0,
			some300:  2.0,
			full10:   0.0,
			expected: "medium",
		},
		{
			name:     "high pressure",
			some10:   25.0,
			some60:   15.0,
			some300:  5.0,
			full10:   2.0,
			expected: "high",
		},
		{
			name:     "critical pressure",
			some10:   60.0,
			some60:   40.0,
			some300:  20.0,
			full10:   15.0,
			expected: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pressure := &SystemPressure{
				Memory: PressureStats{
					Some: PressureMetrics{Avg10: tt.some10, Avg60: tt.some60, Avg300: tt.some300},
					Full: PressureMetrics{Avg10: tt.full10},
				},
			}
			assert.Equal(t, tt.expected, pressure.GetMemoryPressureLevel())
		})
	}
}

func TestIsMemoryPressureHigh(t *testing.T) {
	tests := []struct {
		name     string
		some10   float64
		some60   float64
		some300  float64
		expected bool
	}{
		{
			name:     "no pressure",
			some10:   0.0,
			some60:   0.0,
			some300:  0.0,
			expected: false,
		},
		{
			name:     "low pressure",
			some10:   5.0,
			some60:   2.0,
			some300:  0.5,
			expected: false,
		},
		{
			name:     "high avg10",
			some10:   15.0,
			some60:   2.0,
			some300:  0.5,
			expected: true,
		},
		{
			name:     "high avg60",
			some10:   5.0,
			some60:   8.0,
			some300:  0.5,
			expected: true,
		},
		{
			name:     "high avg300",
			some10:   5.0,
			some60:   2.0,
			some300:  2.0,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pressure := &SystemPressure{
				Memory: PressureStats{
					Some: PressureMetrics{Avg10: tt.some10, Avg60: tt.some60, Avg300: tt.some300},
				},
			}
			assert.Equal(t, tt.expected, pressure.IsMemoryPressureHigh())
		})
	}
}
