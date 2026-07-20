package style

import "unicode"

// wideRanges lists Unicode blocks that monospace terminals render as two
// cells: East Asian Wide and Fullwidth blocks plus the common emoji planes.
var wideRanges = [][2]rune{
	{0x1100, 0x115F}, {0x2329, 0x232A}, {0x2E80, 0x303E}, {0x3041, 0x33FF},
	{0x3400, 0x4DBF}, {0x4E00, 0x9FFF}, {0xA000, 0xA4CF}, {0xA960, 0xA97F},
	{0xAC00, 0xD7A3}, {0xF900, 0xFAFF}, {0xFE10, 0xFE19}, {0xFE30, 0xFE6F},
	{0xFF00, 0xFF60}, {0xFFE0, 0xFFE6}, {0x1F300, 0x1F64F}, {0x1F680, 0x1F6FF},
	{0x1F900, 0x1FAFF}, {0x20000, 0x2FFFD}, {0x30000, 0x3FFFD},
}

func RuneWidth(character rune) int {
	if unicode.In(character, unicode.Mn, unicode.Me, unicode.Cf) {
		return 0
	}
	for _, span := range wideRanges {
		if character >= span[0] && character <= span[1] {
			return 2
		}
	}
	return 1
}

func DisplayWidth(value string) int {
	width := 0
	for _, character := range value {
		width += RuneWidth(character)
	}
	return width
}
