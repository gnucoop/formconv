package formats

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/extrame/xls"
	"github.com/tealeg/xlsx"
)

type XlsForm struct {
	Survey  []SurveyRow
	Choices []ChoicesRow
}
type SurveyRow struct {
	Type, Name, Label,
	Relevant, Constraint, ConstraintMessage, Calculation, Required, RepeatCount string
	LineNum int
}
type ChoicesRow struct {
	ListName, Name, Label string
	LineNum               int
}

// Defines which sheets/columns to read from an excel file.
// Names must appear in the same order as the fields of XlsForm.
var sheetInfos = []sheetInfo{
	{
		name:      "survey",
		mandatory: true,
		columns: []columnInfo{
			{name: "type", mandatory: true},
			{name: "name", mandatory: true},
			{name: "label", mandatory: true},
			{name: "relevant"},
			{name: "constraint"},
			{name: "constraint_message"},
			{name: "calculation"},
			{name: "required"},
			{name: "repeat_count"},
		},
	}, {
		name:      "choices",
		mandatory: true,
		columns: []columnInfo{
			{name: "list name", mandatory: true},
			{name: "name", mandatory: true},
			{name: "label", mandatory: true},
		},
	},
}

type sheetInfo struct {
	name      string
	mandatory bool
	columns   []columnInfo
}
type columnInfo struct {
	name      string
	mandatory bool
}

type File interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

func DecXls(f File, ext string, size int64) (*XlsForm, error) {
	wb, err := NewWorkBook(f, ext, size)
	if err != nil {
		return nil, err
	}

	var form XlsForm
	formVal := reflect.ValueOf(&form).Elem()
	for s, sheetInfo := range sheetInfos {
		rows := wb.Rows(sheetInfo.name)
		if rows == nil && sheetInfo.mandatory {
			return nil, fmt.Errorf("Missing mandatory sheet %q.", sheetInfo.name)
		}
		if rows == nil {
			continue // not mandatory, skip
		}
		headIndex := firstNonempty(rows)
		if headIndex == -1 {
			return nil, fmt.Errorf("Empty sheet %q.", sheetInfo.name)
		}
		head := rows[headIndex]
		colIndices := make([]int, len(sheetInfo.columns))
		for j, colInfo := range sheetInfo.columns {
			colIndices[j] = columnIndex(head, colInfo.name)
			if colInfo.name == "list name" && colIndices[j] == -1 {
				// According to the docs, the column should be called "list name",
				// but it appears as "list_name" in files generated by the Kobo Toolbox.
				colIndices[j] = columnIndex(head, "list_name")
			}
			if colIndices[j] == -1 && colInfo.mandatory {
				return nil, fmt.Errorf("Column %q in sheet %q is mandatory.", colInfo.name, sheetInfo.name)
			}
		}
		destSlice := formVal.Field(s)
		for i := headIndex + 1; i < len(rows); i++ {
			row := rows[i]
			if isEmpty(row) {
				continue
			}
			destRow := reflect.New(destSlice.Type().Elem()).Elem()
			destRow.FieldByName("LineNum").Set(reflect.ValueOf(i + 1))
			for j := range sheetInfo.columns {
				if colIndices[j] != -1 {
					destRow.Field(j).Set(reflect.ValueOf(row[colIndices[j]]))
				}
			}
			destSlice.Set(reflect.Append(destSlice, destRow))
		}
	}
	return &form, nil
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
	return DecXls(f, filepath.Ext(fileName), stat.Size())
}

type WorkBook interface {
	Rows(sheetName string) [][]string
}

type xlsxWorkBook struct {
	xlsx.File
}

func (wb *xlsxWorkBook) Rows(sheetName string) [][]string {
	sheet, ok := wb.Sheet[sheetName]
	if !ok {
		return nil
	}
	rows := make([][]string, sheet.MaxRow+1)
	numCols := sheet.MaxCol + 1
	for i := range rows {
		rows[i] = make([]string, numCols)
		for j := range rows[i] {
			rows[i][j] = sheet.Cell(i, j).Value
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
	for i := range rows {
		rows[i] = make([]string, numCols)
		row := sheet.Row(i)
		if row == nil {
			continue
		}
		for j := range rows[i] {
			rows[i][j] = row.Col(j)
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

func columnIndex(row []string, name string) int {
	for i, cell := range row {
		if cell == name {
			return i
		}
	}
	name = name + "::English (en)"
	for i, cell := range row {
		if cell == name {
			return i
		}
	}
	return -1
}

func ListLanguages(rows [][]string) map[string]bool {
	headIndex := firstNonempty(rows)
	if headIndex == -1 {
		return nil
	}
	langs := make(map[string]bool)
	for _, cell := range rows[headIndex] {
		_, lang := splitLang(cell)
		if lang != "en" {
			langs[lang] = true
		}
	}
	return langs
}

func splitLang(cell string) (name, lang string) {
	name = cell
	lang = "en"

	i := strings.Index(name, "::")
	if i == -1 {
		return
	}
	l := strings.LastIndexByte(cell, '(')
	r := strings.LastIndexByte(cell, ')')
	if l == -1 || r == -1 || l > r || l < i {
		return
	}
	name = cell[0:i]
	lang = cell[l+1 : r]
	return
}

func Translation(rows [][]string, targetLang string) map[string]string {
	headIndex := firstNonempty(rows)
	if headIndex == -1 {
		return nil
	}
	head := rows[headIndex]
	translation := make(map[string]string)
	for en, cell := range head {
		name, sourceLang := splitLang(cell)
		if name == "" || sourceLang != "en" {
			continue
		}
		tr := translationIndex(head, name, targetLang)
		if tr == -1 {
			continue
		}
		for j := headIndex + 1; j < len(rows); j++ {
			row := rows[j]
			if row[en] != "" {
				translation[row[en]] = row[tr]
			}
		}
	}
	return translation
}

func translationIndex(head []string, name, lang string) int {
	prefix := name + "::"
	suffix := "(" + lang + ")"
	for i, cell := range head {
		if strings.HasPrefix(cell, prefix) && strings.HasSuffix(cell, suffix) {
			return i
		}
	}
	return -1
}

func MergeMaps(a, b map[string]string) map[string]string {
	res := make(map[string]string)
	for k, v := range a {
		res[k] = v
	}
	for k, v := range b {
		res[k] = v
	}
	return res
}
