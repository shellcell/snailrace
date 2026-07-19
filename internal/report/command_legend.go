package report

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/shellcell/snailrace/internal/model"
)

func fullCommand(benchmark model.Benchmark) string {
	command := benchmark.Tool.Command
	if len(command) == 0 {
		return ""
	}
	if benchmark.Tool.ShellCommand {
		return safeCommandText(command[0])
	}
	arguments := make([]string, len(command))
	for index, argument := range command {
		arguments[index] = shellQuoteArgument(argument)
	}
	return strings.Join(arguments, " ")
}

func shellQuoteArgument(argument string) string {
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
