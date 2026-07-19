package report

import (
	"fmt"
	"html"
	"io"

	"github.com/shellcell/snailrace/internal/model"
)

func writeHTMLLinkedFiles(writer io.Writer, tool model.ToolInfo) {
	if len(tool.LinkedFiles) == 0 && len(tool.SharedCacheFiles) == 0 {
		return
	}
	fmt.Fprintf(
		writer,
		`<details><summary>%d linked files · %s</summary>`+
			`<div class="scroll"><table><thead><tr><th>Path</th><th>Size</th>`+
			`</tr></thead><tbody>`,
		len(tool.LinkedFiles)+len(tool.SharedCacheFiles),
		formatBytes(float64(tool.LinkedSizeBytes)),
	)
	for _, file := range tool.LinkedFiles {
		fmt.Fprintf(
			writer, `<tr><td class="command">%s</td><td>%s</td></tr>`,
			html.EscapeString(file.Path), formatBytes(float64(file.SizeBytes)),
		)
	}
	for _, path := range tool.SharedCacheFiles {
		fmt.Fprintf(
			writer, `<tr><td class="command">%s</td><td>shared OS cache · excluded</td></tr>`,
			html.EscapeString(path),
		)
	}
	fmt.Fprint(writer, "</tbody></table></div></details>")
}
