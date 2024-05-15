package metrics

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRateMetricResult_SerDeser(t *testing.T) {
	rate := NewRate(time.Second)
	rate.Add(1, 100*time.Millisecond)
	rate.Add(1, 200*time.Millisecond)
	result := GetRateMetricResult(rate)
	raw, err := json.Marshal(result)
	assert.NoError(t, err)
	var deserResult RateMetricResult
	assert.NoError(t, json.Unmarshal(raw, &deserResult))
	assert.Equal(t, rate.Rate(), FromRateMetricResult(deserResult).Rate())
}
