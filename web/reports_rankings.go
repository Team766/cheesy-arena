package web

import (
	"bytes"
	"github.com/Team254/cheesy-arena/game"
	"net/http"
	"strings"
)

// One column of a qualification rankings report. The column sets are game-specific and live in
// reports_rankings_frc.go / reports_rankings_custom.go; the report handlers below are shared.
type rankingColumn struct {
	Header string
	// Width in mm, used by the PDF report only.
	Width float64
	Value func(ranking game.Ranking) string
}

// Generates a CSV-formatted report of the qualification rankings.
func (web *Web) rankingsCsvReportHandler(w http.ResponseWriter, r *http.Request) {
	rankings, err := web.arena.Database.GetAllRankings()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	// Don't set the content type as "text/csv", as that will trigger an automatic download in the browser.
	w.Header().Set("Content-Type", "text/plain")

	columns := rankingCsvColumns()
	headers := make([]string, len(columns))
	for i, column := range columns {
		headers[i] = column.Header
	}

	var buf bytes.Buffer
	buf.WriteString(strings.Join(headers, ","))
	buf.WriteString("\n")
	for _, ranking := range rankings {
		values := make([]string, len(columns))
		for i, column := range columns {
			values[i] = column.Value(ranking)
		}
		buf.WriteString(strings.Join(values, ","))
		buf.WriteString("\n")
	}
	buf.WriteString("\n")

	if _, err := w.Write(buf.Bytes()); err != nil {
		handleWebErr(w, err)
		return
	}
}

// Generates a PDF-formatted report of the qualification rankings.
func (web *Web) rankingsPdfReportHandler(w http.ResponseWriter, r *http.Request) {
	rankings, err := web.arena.Database.GetAllRankings()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	columns := rankingPdfColumns()
	rowHeight := 6.5

	pdf := newReportPdf()
	pdf.AddPage()

	// Render table header row.
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(195, rowHeight, "Team Standings - "+web.arena.EventSettings.Name, "", 1, "C", false, 0, "")
	for i, column := range columns {
		lineBreak := 0
		if i == len(columns)-1 {
			lineBreak = 1
		}
		pdf.CellFormat(column.Width, rowHeight, column.Header, "1", lineBreak, "C", true, 0, "")
	}

	for _, ranking := range rankings {
		// Render ranking info row. The rank itself is bold; everything else is not.
		for i, column := range columns {
			if i == 0 {
				pdf.SetFont("Arial", "B", 10)
			} else if i == 1 {
				pdf.SetFont("Arial", "", 10)
			}
			lineBreak := 0
			if i == len(columns)-1 {
				lineBreak = 1
			}
			pdf.CellFormat(column.Width, rowHeight, column.Value(ranking), "1", lineBreak, "C", false, 0, "")
		}
	}

	addTimeGeneratedFooter(pdf)

	// Write out the PDF file as the HTTP response.
	w.Header().Set("Content-Type", "application/pdf")
	err = pdf.Output(w)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}
