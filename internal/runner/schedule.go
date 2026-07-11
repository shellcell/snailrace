package runner

import "math/rand"

func balancedSchedule(tools, rounds int, seed int64) [][]int {
	if tools <= 0 || rounds <= 0 {
		return nil
	}
	random := rand.New(rand.NewSource(seed))
	result := make([][]int, 0, rounds)
	for len(result) < rounds {
		base := random.Perm(tools)
		shifts := random.Perm(tools)
		for _, shift := range shifts {
			order := make([]int, tools)
			for position := range order {
				order[position] = base[(shift+position)%tools]
			}
			result = append(result, order)
			if len(result) == rounds {
				return result
			}
		}
	}
	return result
}
