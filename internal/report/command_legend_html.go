package report

import (
	"fmt"
	"html"
	"io"

	"github.com/shellcell/snailrace/internal/model"
)

func writeHTMLCommandLegend(writer io.Writer, report model.Report) {
	fmt.Fprint(writer, `<section><h2>Commands</h2><div class="command-list">`)
	for index, benchmark := range report.Benchmarks {
		fmt.Fprintf(
			writer,
			`<div class="command-item"><strong>%d. %s</strong>`+
				`<code>%s</code></div>`,
			index+1, htmlToolLabel(report, index, true),
			html.EscapeString(fullCommand(benchmark)),
		)
	}
	fmt.Fprint(writer, "</div></section>")
}
