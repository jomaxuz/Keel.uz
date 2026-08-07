package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"

	"github.com/xuri/excelize/v2"
)

// One report shape, three ways out: the screen, a spreadsheet, and (later) the
// accountant's 1C.
//
// Every report in this system — dish analysis, takings, cash, payroll — is the
// same thing underneath: a period, some columns, some rows. Writing that once
// means a new report is a query and a column list, and it arrives on screen and
// in Excel at the same moment. The alternative is what usually happens: a
// screen built first, an export bolted on months later against a second copy of
// the query, and two numbers that disagree by the time anyone notices.
//
// ⚠️ **The export must be the same numbers as the screen, not a second
// derivation of them.** An owner who exports a month, sends it to an
// accountant, and is then asked why the spreadsheet disagrees with the panel
// has been given two answers and no way to tell which is wrong.

// ColKind tells the renderer how to format a column, not what it contains.
type ColKind string

const (
	ColText  ColKind = "text"
	ColInt   ColKind = "int"
	ColMoney ColKind = "money"
	// A share of a whole, stored 0–100 and shown with one decimal.
	ColPercent ColKind = "percent"
	ColDate    ColKind = "date"
)

// Column is one field of a report.
type Column struct {
	Key   string  `json:"key"`
	Title string  `json:"title"`
	Kind  ColKind `json:"kind"`
}

// Report is a table with a title and a period, ready for any renderer.
type Report struct {
	// Used as the file name and the sheet name, so it has to be short and
	// mean something on a desktop full of downloads.
	Title   string           `json:"title"`
	From    string           `json:"from"`
	To      string           `json:"to"`
	Columns []Column         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	// Totals row, when the report has one. Keyed like a row; columns with no
	// entry render blank rather than zero — a blank says "this does not add
	// up", while a 0 claims it adds up to nothing.
	Totals map[string]any `json:"totals,omitempty"`
	// Shown under the title on screen and in cell A2 of the sheet. This is
	// where a report explains what it counted, and every report here has
	// something worth saying — "takings" and "orders placed" are different
	// numbers and the difference has burned this dashboard once already.
	Note string `json:"note,omitempty"`
}

// wantsExcel reports whether the caller asked for a spreadsheet rather than
// JSON. A query parameter rather than an Accept header: this is a link an
// operator clicks, and a link cannot set headers.
func wantsExcel(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("format")), "xlsx")
}

// respondReport writes the report as JSON or as a spreadsheet.
func (h *Handler) respondReport(w http.ResponseWriter, r *http.Request, rep *Report) {
	if !wantsExcel(r) {
		httpx.JSON(w, http.StatusOK, rep)
		return
	}
	if err := writeXLSX(w, rep); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// writeXLSX renders the report as a real .xlsx.
//
// A real spreadsheet rather than CSV, and the reason is specific rather than
// aesthetic: this data is Uzbek text and so'm amounts, and CSV gets both wrong
// in the same place. Excel on a Russian or Uzbek Windows reads a comma as a
// decimal separator, so "Lag'mon, katta" splits into two cells and 92,000
// becomes 92; without a BOM the Cyrillic and the apostrophes arrive as
// mojibake. Every one of those is a silent corruption of a financial document.
func writeXLSX(w http.ResponseWriter, rep *Report) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := sheetName(rep.Title)
	idx, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}
	f.SetActiveSheet(idx)
	// NewFile always makes "Sheet1"; leaving it would hand the accountant a
	// workbook whose first tab is empty.
	if sheet != "Sheet1" {
		_ = f.DeleteSheet("Sheet1")
	}

	title, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	head, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F2F2F2"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	// `#,##0` with a space group separator: so'm are written 92 000 here, and a
	// spreadsheet that shows 92000 is one every reader has to squint at.
	money, _ := f.NewStyle(&excelize.Style{CustomNumFmt: strptr(`# ##0`)})
	percent, _ := f.NewStyle(&excelize.Style{CustomNumFmt: strptr(`0.0"%"`)})
	totals, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		CustomNumFmt: strptr(`# ##0`),
	})

	_ = f.SetCellStr(sheet, "A1", rep.Title)
	_ = f.SetCellStyle(sheet, "A1", "A1", title)
	period := rep.From + " — " + rep.To
	if rep.Note != "" {
		period += " · " + rep.Note
	}
	_ = f.SetCellStr(sheet, "A2", period)

	const first = 4 // row 1 title, row 2 period, row 3 blank
	for c, col := range rep.Columns {
		cell, _ := excelize.CoordinatesToCellName(c+1, first)
		_ = f.SetCellStr(sheet, cell, col.Title)
		_ = f.SetCellStyle(sheet, cell, cell, head)
		// Wide enough to read the header without every column being resized by
		// hand. Not auto-fit: excelize cannot measure text, and a guess from
		// the title length beats a default that truncates every one of them.
		width := float64(len([]rune(col.Title))) + 4
		if width < 12 {
			width = 12
		}
		if width > 42 {
			width = 42
		}
		colName, _ := excelize.ColumnNumberToName(c + 1)
		_ = f.SetColWidth(sheet, colName, colName, width)
	}

	write := func(row int, values map[string]any, style int) {
		for c, col := range rep.Columns {
			v, ok := values[col.Key]
			if !ok || v == nil {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(c+1, row)
			// ⚠️ Numbers are written as numbers, never as pre-formatted text.
			// A column of "92 000" strings looks identical on screen and cannot
			// be summed, sorted or charted — which is most of why somebody
			// asked for a spreadsheet instead of a screenshot.
			_ = f.SetCellValue(sheet, cell, v)
			switch col.Kind {
			case ColMoney, ColInt:
				s := money
				if style != 0 {
					s = style
				}
				_ = f.SetCellStyle(sheet, cell, cell, s)
			case ColPercent:
				_ = f.SetCellStyle(sheet, cell, cell, percent)
			default:
				if style != 0 {
					_ = f.SetCellStyle(sheet, cell, cell, style)
				}
			}
		}
	}

	for i, row := range rep.Rows {
		write(first+1+i, row, 0)
	}
	if len(rep.Totals) > 0 {
		write(first+1+len(rep.Rows)+1, rep.Totals, totals)
	}
	// The header stays visible while an accountant scrolls a thousand rows.
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze: true, Split: false, XSplit: 0, YSplit: first,
		TopLeftCell: fmt.Sprintf("A%d", first+1), ActivePane: "bottomLeft",
	})

	name := fmt.Sprintf("%s-%s.xlsx", fileSlug(rep.Title), time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	// Quoted, because the name may contain a space and an unquoted one is
	// truncated at it by browsers.
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	return f.Write(w)
}

func strptr(s string) *string { return &s }

// sheetName trims a title to what Excel accepts as a tab name.
//
// Excel refuses a sheet name over 31 characters or containing any of : \ / ? *
// [ ] — and refuses the whole file, not the name, so a report titled with a
// date range would produce a workbook that will not open.
func sheetName(title string) string {
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`:\/?*[]`, r) {
			return '-'
		}
		return r
	}, strings.TrimSpace(title))
	runes := []rune(name)
	if len(runes) > 31 {
		runes = runes[:31]
	}
	if len(runes) == 0 {
		return "Hisobot"
	}
	return string(runes)
}

// fileSlug turns a title into something safe for a download name on any OS.
func fileSlug(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "hisobot"
	}
	return s
}
