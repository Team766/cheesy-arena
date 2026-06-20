//go:build custom

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

	w.Header().Set("Content-Type", "text/plain")
	var buf bytes.Buffer
	buf.WriteString("Rank,TeamId,RankingPoints,MatchPoints,AutoPoints,Wins,Losses,Ties,Disqualifications,Played\n")
	for _, ranking := range rankings {
		buf.WriteString(fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
			ranking.Rank,
			ranking.TeamId,
			ranking.RankingPoints,
			ranking.MatchPoints,
			ranking.AutoPoints,
			ranking.Wins,
			ranking.Losses,
			ranking.Ties,
			ranking.Disqualifications,
			ranking.Played,
		))
	}

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

	colWidths := map[string]float64{
		"Rank":   15,
		"Team":   25,
		"RP":     25,
		"Match":  30,
		"Auto":   30,
		"W-L-T":  30,
		"DQ":     20,
		"Played": 20,
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
	pdf.CellFormat(colWidths["Match"], rowHeight, "Match Pts", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["Auto"], rowHeight, "Auto Pts", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["W-L-T"], rowHeight, "W-L-T", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["DQ"], rowHeight, "DQ", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths["Played"], rowHeight, "Played", "1", 1, "C", true, 0, "")

	for _, ranking := range rankings {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(colWidths["Rank"], rowHeight, strconv.Itoa(ranking.Rank), "1", 0, "C", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(colWidths["Team"], rowHeight, strconv.Itoa(ranking.TeamId), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths["RP"], rowHeight, strconv.Itoa(ranking.RankingPoints), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths["Match"], rowHeight, strconv.Itoa(ranking.MatchPoints), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths["Auto"], rowHeight, strconv.Itoa(ranking.AutoPoints), "1", 0, "C", false, 0, "")
		record := fmt.Sprintf("%d-%d-%d", ranking.Wins, ranking.Losses, ranking.Ties)
		pdf.CellFormat(colWidths["W-L-T"], rowHeight, record, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths["DQ"], rowHeight, strconv.Itoa(ranking.Disqualifications), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths["Played"], rowHeight, strconv.Itoa(ranking.Played), "1", 1, "C", false, 0, "")
	}

	addTimeGeneratedFooter(pdf)

	w.Header().Set("Content-Type", "application/pdf")
	err = pdf.Output(w)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}
