//go:build !custom

package web

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
)

// Generates a CSV-formatted report of the qualification rankings.
func (web *Web) rankingsCsvReportHandler(w http.ResponseWriter, r *http.Request) {
	rankings, err := web.arena.Database.GetAllRankings()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	// Don't set the content type as "text/csv", as that will trigger an automatic download in the browser.
	w.Header().Set("Content-Type", "text/plain")
	template, err := web.parseFiles("templates/rankings.csv")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	var buf bytes.Buffer
	err = template.ExecuteTemplate(&buf, "rankings.csv", rankings)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	// Strip out carriage returns to ensure consistent behavior across platforms.
	cleaned := bytes.ReplaceAll(buf.Bytes(), []byte("\r"), []byte(""))
	if _, err := w.Write(cleaned); err != nil {
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

	// The widths of the table columns in mm, stored here so that they can be referenced for each row.
	colWidths := map[string]float64{
		"Rank":      13,
		"Team":      20,
		"RP":        24,
		"Match":     24,
		"Auto Fuel": 24,
		"Tower":     24,
		"W-L-T":     26,
		"DQ":        20,
		"Played":    20,
	}
	rowHeight := 6.5

	pdf := newReportPdf()
	pdf.AddPage()

	// Render table header row.
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(195, rowHeight, "Team Standings - "+web.arena.EventSettings.Name, "", 1, "C", false, 0, "")
	pdf.CellFormat(colWidths["Rank"], rowHeight, "Rank", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["Team"], rowHeight, "Team", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["RP"], rowHeight, "RP", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["Match"], rowHeight, "Match", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["Auto Fuel"], rowHeight, "Auto Fuel", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["Tower"], rowHeight, "Tower", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["W-L-T"], rowHeight, "W-L-T", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["DQ"], rowHeight, "DQ", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["Played"], rowHeight, "Played", "1", 1, "C", true, 0, "")
	for _, ranking := range rankings {
		// Render ranking info row.
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(colWidths["Rank"], rowHeight, strconv.Itoa(ranking.Rank), "1", 0, "C", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(colWidths["Team"], rowHeight, strconv.Itoa(ranking.TeamId), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths["RP"], rowHeight, strconv.Itoa(ranking.RankingPoints), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths["Match"], rowHeight, strconv.Itoa(ranking.MatchPoints), "1", 0, "C", false, 0, "")
		pdf.CellFormat(
			colWidths["Auto Fuel"], rowHeight, strconv.Itoa(ranking.AutoFuelPoints), "1", 0, "C", false, 0, "",
		)
		pdf.CellFormat(colWidths["Tower"], rowHeight, strconv.Itoa(ranking.TowerPoints), "1", 0, "C", false, 0, "")
		record := fmt.Sprintf("%d-%d-%d", ranking.Wins, ranking.Losses, ranking.Ties)
		pdf.CellFormat(colWidths["W-L-T"], rowHeight, record, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths["DQ"], rowHeight, strconv.Itoa(ranking.Disqualifications), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths["Played"], rowHeight, strconv.Itoa(ranking.Played), "1", 1, "C", false, 0, "")
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
