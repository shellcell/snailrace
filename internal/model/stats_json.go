package model

import "encoding/json"

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
