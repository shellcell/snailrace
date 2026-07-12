package runner

import "testing"

func TestBalancedScheduleCounterbalancesCompleteBlock(t *testing.T) {
	const tools = 4
	schedule := BalancedSchedule(tools, tools, 42)
	for tool := 0; tool < tools; tool++ {
		positions := make([]int, tools)
		for _, round := range schedule {
			for position, current := range round {
				if current == tool {
					positions[position]++
				}
			}
		}
		for position, count := range positions {
			if count != 1 {
				t.Fatalf("tool %d appears %d times at position %d", tool, count, position)
			}
		}
	}
}
