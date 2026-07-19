package style

import (
	"strconv"
	"strings"
	"unicode"
)

func SafeText(value string) string {
	var result strings.Builder
	for _, character := range value {
		if !unicode.IsControl(character) {
			result.WriteRune(character)
			continue
		}
		quoted := strconv.QuoteRuneToGraphic(character)
		result.WriteString(quoted[1 : len(quoted)-1])
	}
	return result.String()
}
