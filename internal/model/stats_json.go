package model

import (
	"encoding/json"
	"math"
)

func (stats Stats) MarshalJSON() ([]byte, error) {
	type plain Stats
	var low, high *float64
	if stats.CI95Valid {
		low, high = &stats.CI95Low, &stats.CI95High
	}
	return json.Marshal(struct {
		plain
		CI95Low  *float64 `json:"ci95_low,omitempty"`
		CI95High *float64 `json:"ci95_high,omitempty"`
	}{plain(stats), low, high})
}

func (stats *Stats) UnmarshalJSON(data []byte) error {
	type plain Stats
	decoded := struct {
		plain
		CI95Low  *float64 `json:"ci95_low"`
		CI95High *float64 `json:"ci95_high"`
	}{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*stats = Stats(decoded.plain)
	if !stats.CI95Valid {
		return nil
	}
	if decoded.CI95Low == nil || decoded.CI95High == nil {
		if stats.N < 2 {
			stats.CI95Valid = false
			return nil
		}
		margin := stats.StdDev / math.Sqrt(float64(stats.N)) * tCritical95(stats.N-1)
		stats.CI95Low = finiteDifference(stats.Mean, margin)
		stats.CI95High = finiteSum(stats.Mean, margin)
		return nil
	}
	stats.CI95Low = *decoded.CI95Low
	stats.CI95High = *decoded.CI95High
	return nil
}
