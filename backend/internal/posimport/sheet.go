package posimport

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// Reading the file, whatever the other system produced.

// maxRows is a ceiling on one import. A nomenclature is thousands of lines at
// the outside; a file past this is not one.
const maxRows = 20000

// Sheet is a file reduced to a header and rows.
type Sheet struct {
	Header []string
	Rows   [][]string
}

// Read turns an uploaded file into rows.
//
// ⚠️ **XLSX and CSV both, because the export button decides and the owner does
// not.** iiko offers Excel, Poster offers CSV, and telling somebody mid-migration
// that they exported the wrong kind is the point at which they stop.
func Read(name string, data []byte) (*Sheet, error) {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".xlsx") || strings.HasSuffix(lower, ".xlsm") {
		return readExcel(data)
	}
	return readCSV(data)
}

func readExcel(data []byte) (*Sheet, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("faylni ochib bo'lmadi — Excel yoki CSV bo'lishi kerak")
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("faylda varaq yo'q")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}
	return fromRows(rows)
}

func readCSV(data []byte) (*Sheet, error) {
	if !utf8.Valid(data) {
		// ⚠️ **Windows-1251, and it is the ordinary case.** These systems run on
		// Windows and their CSV export is not UTF-8; read as UTF-8 it produces
		// a header of replacement characters, no column is recognised, and the
		// import looks like it does not understand the file at all.
		data = fromWin1251(data)
	}
	// ⚠️ The separator is a semicolon on a Russian Windows, because the comma is
	// the decimal mark. Sniffed rather than assumed: with the wrong one every
	// row is a single cell and the file appears to have one column.
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = sniffSeparator(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	var rows [][]string
	for len(rows) < maxRows {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// One malformed line does not end an import of four thousand.
			continue
		}
		rows = append(rows, rec)
	}
	return fromRows(rows)
}

func sniffSeparator(data []byte) rune {
	head := data
	if len(head) > 4096 {
		head = head[:4096]
	}
	if bytes.Count(head, []byte(";")) > bytes.Count(head, []byte(",")) {
		return ';'
	}
	if bytes.Count(head, []byte("\t")) > bytes.Count(head, []byte(",")) {
		return '\t'
	}
	return ','
}

// fromRows finds the header and drops what is above it.
//
// ⚠️ **The header is not row one.** Every one of these exports puts a title, a
// date range and the restaurant's name above the table, and often a blank row
// after. Taking row one as the header names every column after a report title,
// nothing matches, and the file looks unreadable.
func fromRows(rows [][]string) (*Sheet, error) {
	if len(rows) == 0 {
		return nil, errors.New("fayl bo'sh")
	}
	best, bestScore := -1, 0
	// Only the first fifteen: past that we are reading data as headings.
	for i, row := range rows {
		if i > 15 {
			break
		}
		score := 0
		for _, cell := range row {
			if fieldOf(cell) != "" {
				score++
			}
		}
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	if best < 0 || bestScore == 0 {
		return nil, errors.New(
			"ustun sarlavhalari tanilmadi — faylning birinchi qatorida " +
				"nom, birlik, narx kabi sarlavhalar bo'lishi kerak")
	}
	sheet := &Sheet{Header: rows[best]}
	for _, row := range rows[best+1:] {
		if blank(row) {
			continue
		}
		sheet.Rows = append(sheet.Rows, row)
		if len(sheet.Rows) >= maxRows {
			break
		}
	}
	return sheet, nil
}

func blank(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// fromWin1251 decodes the one legacy encoding these exports actually use.
//
// ⚠️ Written out rather than pulled in as a dependency: it is one lookup table
// for the Cyrillic range, and every byte above 0x7F maps to exactly one rune.
func fromWin1251(data []byte) []byte {
	var b strings.Builder
	b.Grow(len(data) * 2)
	for _, c := range data {
		switch {
		case c < 0x80:
			b.WriteByte(c)
		case c >= 0xC0:
			// А–я, one contiguous block.
			b.WriteRune(rune(0x0410 + int(c) - 0xC0))
		default:
			b.WriteRune(win1251High[c-0x80])
		}
	}
	return []byte(b.String())
}

// The 0x80–0xBF range, which holds the punctuation and the stray letters.
var win1251High = [64]rune{
	'Ђ', 'Ѓ', '‚', 'ѓ', '„', '…', '†', '‡', '€', '‰', 'Љ', '‹', 'Њ', 'Ќ', 'Ћ', 'Џ',
	'ђ', '‘', '’', '“', '”', '•', '–', '—', ' ', '™', 'љ', '›', 'њ', 'ќ', 'ћ', 'џ',
	' ', 'Ў', 'ў', 'Ј', '¤', 'Ґ', '¦', '§', 'Ё', '©', 'Є', '«', '¬', '­', '®', 'Ї',
	'°', '±', 'І', 'і', 'ґ', 'µ', '¶', '·', 'ё', '№', 'є', '»', 'ј', 'Ѕ', 'ѕ', 'ї',
}
