package formats

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/extrame/xls"
	"github.com/tealeg/xlsx"
)

type XlsForm struct {
	Survey   []SurveyRow
	Choices  []ChoicesRow
	Settings []SettingsRow
	Tables   map[string][][]string
	LangSet  map[string]bool
}

type Row struct {
	cells   map[string]string
	LineNum int
}

func makeRow(keyIsValid func(string) bool, keyVals ...string) Row {
	var row Row
	row.cells = make(map[string]string)
	for k, v := 0, 1; v < len(keyVals); k, v = k+2, v+2 {
		key := keyVals[k]
		if !keyIsValid(key) {
			panic(fmt.Sprintf("Invalid column %q in row", key))
		}
		row.cells[key] = keyVals[v]
	}
	return row
}

func (r Row) langCell(name, lang string) string {
	if lang == "" {
		return r.cells[name]
	}
	return r.cells[name+"::"+lang]
}

type SurveyRow struct {
	Row
	// Type is kept here as an optimization,
	// to avoid accessing Row.cells["type"] too frequently.
	Type string
}

func MakeSurveyRow(keyVals ...string) SurveyRow {
	row := makeRow(isSurveyCol, keyVals...)
	return SurveyRow{row, row.cells["type"]}
}

var surveyCols = map[string]bool{
	"type": true, "name": true, "label": true, "hint": true, "appearance": true,
	"relevant": true, "permissions_relevant": true, "constraint": true, "constraint_message": true,
	"calculation": true, "required": true, "required_message": true, "repeat_count": true,
	"choice_filter": true, "default": true, "readonly": true, "parameters": true,
}

func isSurveyCol(name string) bool {
	return surveyCols[name] ||
		strings.HasPrefix(name, "label") || strings.HasPrefix(name, "required_message") ||
		strings.HasPrefix(name, "hint") || strings.HasPrefix(name, "constraint_message")
}

func (r SurveyRow) Name() string                       { return r.cells["name"] }
func (r SurveyRow) Label(lang string) string           { return r.langCell("label", lang) }
func (r SurveyRow) Hint(lang string) string            { return r.langCell("hint", lang) }
func (r SurveyRow) Relevant() string                   { return r.cells["relevant"] }
func (r SurveyRow) PermissionsRelevant() string        { return r.cells["permissions_relevant"] }
func (r SurveyRow) Default() string                    { return r.cells["default"] }
func (r SurveyRow) ReadOnly() string                   { return r.cells["readonly"] }
func (r SurveyRow) Constraint() string                 { return r.cells["constraint"] }
func (r SurveyRow) ConstraintMsg(lang string) string   { return r.langCell("constraint_message", lang) }
func (r SurveyRow) Calculation() string                { return r.cells["calculation"] }
func (r SurveyRow) Required() string                   { return r.cells["required"] }
func (r SurveyRow) RequiredMessage(lang string) string { return r.langCell("required_message", lang) }
func (r SurveyRow) RepeatCount() string                { return r.cells["repeat_count"] }
func (r SurveyRow) ChoiceFilter() string               { return r.cells["choice_filter"] }
func (r SurveyRow) Parameters() string                 { return r.cells["parameters"] }
func (r SurveyRow) Appearance() string                 { return r.cells["appearance"] }

type ChoicesRow struct{ Row }

func MakeChoicesRow(keyVals ...string) ChoicesRow {
	return ChoicesRow{makeRow(isChoicesCol, keyVals...)}
}

func isChoicesCol(name string) bool {
	return name == "list name" || name == "name" || strings.HasPrefix(name, "label")
}

func (r ChoicesRow) ListName() string         { return r.cells["list name"] }
func (r ChoicesRow) Name() string             { return r.cells["name"] }
func (r ChoicesRow) Label(lang string) string { return r.langCell("label", lang) }
func (r ChoicesRow) UserDefCells() map[string]string {
	ud := make(map[string]string)
	for k, v := range r.cells {
		if !isChoicesCol(k) {
			ud[k] = v
		}
	}
	return ud
}

type SettingsRow struct{ Row }

func MakeSettingsRow(keyVals ...string) SettingsRow {
	return SettingsRow{makeRow(isSettingsCol, keyVals...)}
}

func isSettingsCol(name string) bool {
	return name == "tag label" || name == "tag value"
}

func (r SettingsRow) TagLabel() string { return r.cells["tag label"] }
func (r SettingsRow) TagValue() string { return r.cells["tag value"] }

type File interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

func DecXlsform(wb WorkBook) (*XlsForm, error) {
	var form XlsForm
	for _, sheetName := range []string{"survey", "choices", "settings"} {
		rows := wb.Rows(sheetName)
		canonicalize(rows)
		headIndex := firstNonempty(rows)
		if headIndex == -1 && sheetName == "settings" {
			continue // ok, settings sheet is optional
		}
		if headIndex == -1 {
			return nil, fmt.Errorf("Mandatory sheet %q missing or empty.", sheetName)
		}
		head := rows[headIndex]
		if sheetName == "survey" || sheetName == "choices" {
			form.LangSet = mergeSets(form.LangSet, langSet(head))
		}
		for i := headIndex + 1; i < len(rows); i++ {
			if isEmpty(rows[i]) {
				continue
			}
			var destRow Row
			destRow.cells = make(map[string]string)
			destRow.LineNum = i + 1
			for j, cell := range rows[i] {
				colName := head[j]
				if colName != "" && cell != "" {
					destRow.cells[colName] = cell
				}
			}
			switch sheetName {
			case "survey":
				form.Survey = append(form.Survey, SurveyRow{destRow, destRow.cells["type"]})
			case "choices":
				form.Choices = append(form.Choices, ChoicesRow{destRow})
			case "settings":
				form.Settings = append(form.Settings, SettingsRow{destRow})
			}
		}
	}
	form.Tables = make(map[string][][]string)
	for _, row := range form.Survey {
		if row.Type == "table" {
			name := row.Name()
			tab := wb.Rows(name)
			if tab == nil {
				return nil, fmt.Errorf("No sheet for table %q.", name)
			}
			form.Tables[name] = tab
		}
	}
	return &form, nil
}

func canonicalize(rows [][]string) {
	for _, row := range rows {
		for i, cell := range row {
			switch {
			case cell == "list_name":
				row[i] = "list name"
			case cell == "begin_group":
				row[i] = "begin group"
			case cell == "end_group":
				row[i] = "end group"
			case strings.HasPrefix(cell, "select one"):
				row[i] = strings.Replace(cell, "select one", "select_one", 1)
			case strings.HasPrefix(cell, "select multiple"):
				row[i] = strings.Replace(cell, "select multiple", "select_multiple", 1)
			}
		}
	}
}

func DecXlsFromFile(fileName string) (*XlsForm, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("Couldn't open file: %s", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("Couldn't get file stat: %s", err)
	}
	wb, err := NewWorkBook(f, filepath.Ext(fileName), stat.Size())
	if err != nil {
		return nil, err
	}
	return DecXlsform(wb)
}

type WorkBook interface {
	Rows(sheetName string) [][]string
}

type xlsxWorkBook struct {
	xlsx.File
}

// When we find more than 50 consecutive empty rows in a sheet,
// we assume the rest of the sheet is empty and truncate it.
const maxConsecEmptyRows = 50

func (wb *xlsxWorkBook) Rows(sheetName string) [][]string {
	sheet, ok := wb.Sheet[sheetName]
	if !ok {
		return nil
	}
	rows := make([][]string, sheet.MaxRow+1)
	numCols := sheet.MaxCol + 1
	consecEmptyRows := 0
	for i := range rows {
		rows[i] = make([]string, numCols)
		for j := range rows[i] {
			rows[i][j] = sheet.Cell(i, j).Value
		}
		if isEmpty(rows[i]) {
			consecEmptyRows++
		} else {
			consecEmptyRows = 0
		}
		if consecEmptyRows > maxConsecEmptyRows {
			return rows[0 : i+1]
		}
	}
	return rows
}

type xlsWorkBook struct {
	xls.WorkBook
}

func (wb *xlsWorkBook) Rows(sheetName string) [][]string {
	var sheet *xls.WorkSheet
	for i := 0; i < wb.NumSheets(); i++ {
		if s := wb.GetSheet(i); s.Name == sheetName {
			sheet = s
			break
		}
	}
	if sheet == nil {
		return nil
	}
	rows := make([][]string, sheet.MaxRow+1)
	numCols := 0
	for i := range rows {
		if row := sheet.Row(i); row != nil && row.LastCol()+1 > numCols {
			numCols = row.LastCol() + 1
		}
	}
	consecEmptyRows := 0
	for i := range rows {
		rows[i] = make([]string, numCols)
		row := sheet.Row(i)
		if row == nil {
			continue
		}
		for j := range rows[i] {
			rows[i][j] = row.Col(j)
		}
		if isEmpty(rows[i]) {
			consecEmptyRows++
		} else {
			consecEmptyRows = 0
		}
		if consecEmptyRows > maxConsecEmptyRows {
			return rows[0 : i+1]
		}
	}
	return rows
}

func NewWorkBook(f File, ext string, size int64) (WorkBook, error) {
	switch ext {
	case ".xls":
		wb, err := xls.OpenReader(f, "utf-8")
		if err != nil {
			return nil, err
		}
		return &xlsWorkBook{*wb}, nil
	case ".xlsx":
		wb, err := xlsx.OpenReaderAt(f, size)
		if err != nil {
			return nil, err
		}
		return &xlsxWorkBook{*wb}, nil
	default:
		return nil, fmt.Errorf("Unsupported excel file type %s.", ext)
	}
}

func isEmpty(row []string) bool {
	for _, cell := range row {
		if cell != "" {
			return false
		}
	}
	return true
}

func firstNonempty(rows [][]string) int {
	for i, row := range rows {
		if !isEmpty(row) {
			return i
		}
	}
	return -1
}

func getLang(cell string) string {
	i := strings.Index(cell, "::")
	if i == -1 {
		return ""
	}
	return cell[i+2:]
}

func langSet(head []string) map[string]bool {
	langs := make(map[string]bool)
	for _, cell := range head {
		lang := getLang(cell)
		if lang != "" {
			langs[lang] = true
		}
	}
	return langs
}

func mergeSets(a, b map[string]bool) map[string]bool {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	res := make(map[string]bool)
	for k, v := range a {
		res[k] = v
	}
	for k, v := range b {
		res[k] = v
	}
	return res
}
