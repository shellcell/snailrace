package report

import (
	"fmt"
	"html"
	"io"

	"perftool/internal/model"
)

func writeHTMLLinkedFiles(writer io.Writer, tool model.ToolInfo) {
	if len(tool.LinkedFiles) == 0 {
		return
	}
	fmt.Fprintf(
		writer,
		`<details><summary>%d linked files · %s</summary>`+
			`<div class="scroll"><table><thead><tr><th>Path</th><th>Size</th>`+
			`</tr></thead><tbody>`,
		len(tool.LinkedFiles), formatBytes(float64(tool.LinkedSizeBytes)),
	)
	for _, file := range tool.LinkedFiles {
		fmt.Fprintf(
			writer, `<tr><td class="command">%s</td><td>%s</td></tr>`,
			html.EscapeString(file.Path), formatBytes(float64(file.SizeBytes)),
		)
	}
	fmt.Fprint(writer, "</tbody></table></div></details>")
}
