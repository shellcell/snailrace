package style

type Color struct {
	Hex     string
	R, G, B uint8
}

var toolPalette = [...]Color{
	{"#88c0d0", 136, 192, 208}, {"#ebcb8b", 235, 203, 139},
	{"#bf616a", 191, 97, 106}, {"#a3be8c", 163, 190, 140},
	{"#b48ead", 180, 142, 173}, {"#d08770", 208, 135, 112},
	{"#5e81ac", 94, 129, 172}, {"#8fbcbb", 143, 188, 187},
	{"#e5e9f0", 229, 233, 240}, {"#c06c84", 192, 108, 132},
	{"#93c5fd", 147, 197, 253}, {"#f0a6ca", 240, 166, 202},
}

func Tool(index int) Color {
	index %= len(toolPalette)
	if index < 0 {
		index += len(toolPalette)
	}
	return toolPalette[index]
}
