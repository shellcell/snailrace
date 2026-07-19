package style

import (
	"strconv"
	"strings"
	"unicode"
)

func CommandText(command []string, shell bool) string {
	if len(command) == 0 {
		return ""
	}
	if shell {
		return safeCommandText(command[0])
	}
	arguments := make([]string, len(command))
	for index, argument := range command {
		arguments[index] = quoteArgument(argument)
	}
	return strings.Join(arguments, " ")
}

func quoteArgument(argument string) string {
	if strings.IndexFunc(argument, unicode.IsControl) >= 0 {
		return strconv.QuoteToGraphic(argument)
	}
	if argument != "" && strings.IndexFunc(argument, func(character rune) bool {
		return !(unicode.IsLetter(character) || unicode.IsDigit(character) ||
			strings.ContainsRune("_@%+=:,./-", character))
	}) < 0 {
		return argument
	}
	return "'" + strings.ReplaceAll(argument, "'", `'"'"'`) + "'"
}

func safeCommandText(command string) string {
	if strings.IndexFunc(command, unicode.IsControl) >= 0 {
		return strconv.QuoteToGraphic(command)
	}
	return command
}
