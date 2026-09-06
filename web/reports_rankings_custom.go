//go:build custom

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena/game"
	"net/http"
	"strconv"
	"strings"
)

func (web *Web) rankingsCsvReportHandler(w http.ResponseWriter, r *http.Request) {
	rankings, err := web.arena.Database.GetAllRankings()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	cfg := game.GetActiveConfig()
	var headers []string
	headers = append(headers, "Rank", "Team", "RP")
	if cfg != nil {
		for _, tb := range cfg.RankingTiebreakers {
			headers = append(headers, tb.Metric)
		}
	}
	headers = append(headers, "W-L-T", "DQ", "Played")

	var rows []string
	rows = append(rows, strings.Join(headers, ","))

	for _, ranking := range rankings {
		var row []string
		row = append(row, strconv.Itoa(ranking.Rank))
		row = append(row, strconv.Itoa(ranking.TeamId))
		row = append(row, strconv.Itoa(ranking.RankingPoints))
		if cfg != nil {
			for _, tb := range cfg.RankingTiebreakers {
				val := ranking.Tiebreakers[tb.Metric]
				row = append(row, strconv.Itoa(val))
			}
		}
		record := fmt.Sprintf("%d-%d-%d", ranking.Wins, ranking.Losses, ranking.Ties)
		row = append(row, record)
		row = append(row, strconv.Itoa(ranking.Disqualifications))
		row = append(row, strconv.Itoa(ranking.Played))
		rows = append(rows, strings.Join(row, ","))
	}

	csvOutput := strings.Join(rows, "\n") + "\n"
	if _, err := w.Write([]byte(csvOutput)); err != nil {
		handleWebErr(w, err)
		return
	}
}

func (web *Web) rankingsPdfReportHandler(w http.ResponseWriter, r *http.Request) {
	rankings, err := web.arena.Database.GetAllRankings()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	cfg := game.GetActiveConfig()
	rowHeight := 6.5

	pdf := newReportPdf()
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(195, rowHeight, "Team Standings - "+web.arena.EventSettings.Name, "", 1, "C", false, 0, "")

	var cols []string
	cols = append(cols, "Rank", "Team", "RP")
	if cfg != nil {
		for _, tb := range cfg.RankingTiebreakers {
			cols = append(cols, tb.Metric)
		}
	}
	cols = append(cols, "W-L-T", "DQ", "Played")

	widthPerCol := 195.0 / float64(len(cols))

	for _, colName := range cols {
		pdf.CellFormat(widthPerCol, rowHeight, colName, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	for _, ranking := range rankings {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(widthPerCol, rowHeight, strconv.Itoa(ranking.Rank), "1", 0, "C", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(widthPerCol, rowHeight, strconv.Itoa(ranking.TeamId), "1", 0, "C", false, 0, "")
		pdf.CellFormat(widthPerCol, rowHeight, strconv.Itoa(ranking.RankingPoints), "1", 0, "C", false, 0, "")

		if cfg != nil {
			for _, tb := range cfg.RankingTiebreakers {
				val := ranking.Tiebreakers[tb.Metric]
				pdf.CellFormat(widthPerCol, rowHeight, strconv.Itoa(val), "1", 0, "C", false, 0, "")
			}
		}

		record := fmt.Sprintf("%d-%d-%d", ranking.Wins, ranking.Losses, ranking.Ties)
		pdf.CellFormat(widthPerCol, rowHeight, record, "1", 0, "C", false, 0, "")
		pdf.CellFormat(widthPerCol, rowHeight, strconv.Itoa(ranking.Disqualifications), "1", 0, "C", false, 0, "")
		pdf.CellFormat(widthPerCol, rowHeight, strconv.Itoa(ranking.Played), "1", 1, "C", false, 0, "")
	}

	addTimeGeneratedFooter(pdf)

	w.Header().Set("Content-Type", "application/pdf")
	err = pdf.Output(w)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

func isCustomBuild() bool {
	return true
}
