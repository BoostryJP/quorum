package miner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMinerDefautConfig(t *testing.T) {
	type int_data struct {
		actual   uint64
		expected uint64
	}
	var testData = map[string]int_data{
		"DefaultConfig.GasFloor":          {DefaultConfig.GasFloor, 700000000},
		"DefaultConfig.GasCeil":           {DefaultConfig.GasCeil, 800000000},
		"DefaultConfig.Recommit":          {uint64(DefaultConfig.Recommit), uint64(3 * time.Second)},
		"DefaultConfig.NewPayloadTimeout": {uint64(DefaultConfig.NewPayloadTimeout), uint64(300 * time.Millisecond)},
	}
	for k, v := range testData {
		assert.Equal(t, v.expected, v.actual, k+" value mismatch")
	}
}
