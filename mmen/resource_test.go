package mmen

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetResource(t *testing.T) {
	res, err := GetResource()
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Greater(t, res.Timestamp, int64(0))

	// CPU
	assert.GreaterOrEqual(t, res.CPU.CoreCount, int32(1))
	assert.GreaterOrEqual(t, res.CPU.PhysicalCount, int32(1))
	assert.GreaterOrEqual(t, res.CPU.UsagePercent, float64(0))
	assert.LessOrEqual(t, res.CPU.UsagePercent, float64(100))
	assert.Len(t, res.CPU.PerCore, int(res.CPU.CoreCount))

	// Memory
	assert.Greater(t, res.Memory.Total, uint64(0))
	assert.GreaterOrEqual(t, res.Memory.Used, uint64(0))
	assert.GreaterOrEqual(t, res.Memory.UsedPercent, float64(0))
	assert.LessOrEqual(t, res.Memory.UsedPercent, float64(100))

	// Disk
	assert.NotEmpty(t, res.Disks)
	for _, d := range res.Disks {
		assert.NotEmpty(t, d.Path)
		assert.GreaterOrEqual(t, d.UsedPercent, float64(0))
		assert.LessOrEqual(t, d.UsedPercent, float64(100))
	}

	// DiskIO
	assert.NotEmpty(t, res.DiskIOs)

	// Network
	assert.NotEmpty(t, res.Networks)
	for _, n := range res.Networks {
		assert.NotEmpty(t, n.Name)
	}

	// pretty print for manual inspection
	out, _ := json.MarshalIndent(res, "", "  ")
	t.Logf("Resource snapshot:\n%s", out)
}

func TestGetResourceWithSpeed(t *testing.T) {
	res, err := GetResourceWithSpeed(500 * time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, res)

	for _, dio := range res.DiskIOs {
		assert.GreaterOrEqual(t, dio.ReadSpeed, float64(0))
		assert.GreaterOrEqual(t, dio.WriteSpeed, float64(0))
	}

	for _, n := range res.Networks {
		assert.GreaterOrEqual(t, n.SentSpeed, float64(0))
		assert.GreaterOrEqual(t, n.RecvSpeed, float64(0))
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	t.Logf("Resource with speed:\n%s", out)
}

func TestGetResourceWithSpeed_InvalidInterval(t *testing.T) {
	_, err := GetResourceWithSpeed(0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interval must be positive")
}
