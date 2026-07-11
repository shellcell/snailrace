package report

import (
	"fmt"
	"html"
	"io"
	"math"

	"perftool/internal/model"
)

func writeHTMLRanking(writer io.Writer, report model.Report) {
	ranking := calculateRanking(report)
	baselineName := report.Benchmarks[baselineIndex(report)].Tool.Name
	fmt.Fprint(writer, `<section><h2>Category winners</h2><div class="winner-grid">`)
	for _, winner := range ranking.winners {
		winnerIndex := benchmarkIndexByName(report, winner.tool)
		name := htmlToolLabel(report, winnerIndex, winner.tool == baselineName)
		fmt.Fprintf(
			writer,
			`<div class="winner"><small>%s</small><strong>%s</strong>`+
				`<span class="muted">%s</span></div>`,
			html.EscapeString(winner.category), name,
			html.EscapeString(winner.value),
		)
	}
	fmt.Fprintf(
		writer,
		`</div><h2>Overall ranking</h2><p class="muted">Comparison baseline: `+
			`%s. Lower balanced index is better. Select a heading to sort.</p>`,
		htmlToolLabel(report, baselineIndex(report), true),
	)
	fmt.Fprint(writer, `<div class="scroll">`+
		`<table class="ranking" id="ranking"><thead><tr>`+
		rankingHeader(0, "Rank", "number")+rankingHeader(1, "Tool", "text")+
		rankingHeader(2, "Balanced index", "number"))
	fmt.Fprint(writer, rankingHeader(3, ranking.primaryLabel, "number"))
	fmt.Fprint(writer, rankingHeader(4, "CPU cost", "number")+
		rankingHeader(5, "RAM aggregate", "number")+
		rankingHeader(6, "Linked size", "number")+
		`</tr></thead><tbody>`)
	for _, row := range ranking.rows {
		ramValue := "N/A"
		if ranking.ramAvailable {
			ramValue = rankingHTMLCell(
				formatBytes(row.ramValue), row.ramScore, row.ramRank,
			)
		}
		class := ""
		if row.overallRank == 1 {
			class = ` class="rank-one"`
		}
		fmt.Fprintf(
			writer,
			`<tr><td%s data-sort-value="%d">#%d</td>`+
				`<td data-sort-value="%s">%s</td>`+
				`<td data-sort-value="%g">%s</td>`+
				`<td data-sort-value="%g">%s</td>`+
				`<td data-sort-value="%g">%s</td>`+
				`<td data-sort-value="%g">%s</td>`+
				`<td data-sort-value="%g">%s</td></tr>`,
			class, row.overallRank, row.overallRank,
			html.EscapeString(report.Benchmarks[row.benchmark].Tool.Name),
			rankingHTMLTool(report, row.benchmark),
			row.overallScore,
			rankingHTMLCell(
				formatScore(row.overallScore), row.overallScore/ranking.bestOverall,
				row.overallRank,
			),
			row.primaryValue,
			rankingHTMLCell(
				ranking.primaryUnit(row.primaryValue), row.primaryScore,
				row.primaryRank,
			),
			row.cpuValue,
			rankingHTMLCell(cpuRankingValue(report, row), row.cpuScore, row.cpuRank),
			row.ramValue,
			ramValue,
			row.footprintValue,
			rankingHTMLCell(
				formatBytes(row.footprintValue), row.footprintScore,
				row.footprintRank,
			),
		)
	}
	fmt.Fprint(writer, "</tbody></table></div></section>"+rankingSortScript)
}

func rankingHeader(index int, label, kind string) string {
	return fmt.Sprintf(
		`<th><button type="button" data-column="%d" data-kind="%s">%s ↕</button></th>`,
		index, kind, html.EscapeString(label),
	)
}

const rankingSortScript = `<script>(()=>{const t=document.getElementById('ranking');if(!t)return;` +
	`for(const b of t.querySelectorAll('th button'))b.addEventListener('click',()=>{` +
	`const i=+b.dataset.column,n=b.dataset.kind==='number',body=t.tBodies[0],` +
	`rows=[...body.rows],asc=b.dataset.order!=='asc';rows.sort((a,c)=>{` +
	`let x=a.cells[i].dataset.sortValue||a.cells[i].textContent,` +
	`y=c.cells[i].dataset.sortValue||c.cells[i].textContent;if(n){x=+x;y=+y}` +
	`return(x<y?-1:x>y?1:0)*(asc?1:-1)});for(const r of rows)body.appendChild(r);` +
	`b.dataset.order=asc?'asc':'desc'})})()</script>`

func rankingHTMLTool(report model.Report, index int) string {
	return htmlToolLabel(report, index, true)
}

func rankingHTMLCell(value string, score float64, rank int) string {
	if math.IsInf(score, 1) || math.IsNaN(score) {
		return "N/A"
	}
	class := "bad"
	if rank == 1 {
		class = "good"
	}
	return fmt.Sprintf(
		"<strong class=\"%s\">%s</strong>"+
			"<span class=\"delta muted\">%s from best · #%d</span>",
		class, html.EscapeString(value), formatSignedPercent((score-1)*100), rank,
	)
}
