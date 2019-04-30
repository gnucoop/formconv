package formats

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/extrame/xls"
	"github.com/tealeg/xlsx"
)

type XlsForm struct {
	Survey  []SurveyRow
	Choices []ChoicesRow
}
type SurveyRow struct {
	Type, Name, Label,
	Relevant, Constraint, Calculation, Required, RepeatCount string
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

func DecXlsFromFile(fileName string) (*XlsForm, error) {
	wb, err := openWorkBook(fileName)
	if err != nil {
		return nil, err
	}
	defer wb.Close()

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
			colIndices[j] = indexOfString(head, colInfo.name)
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

type workBook interface {
	Rows(sheetName string) [][]string
	Close() error
}

type xlsxWorkBook struct {
	xlsx.File
	io.Closer
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
	io.Closer
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

func openWorkBook(fileName string) (wb workBook, err error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			f.Close()
		}
	}()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	switch ext := filepath.Ext(fileName); ext {
	case ".xls":
		wb, err := xls.OpenReader(f, "utf-8")
		if err != nil {
			return nil, err
		}
		return &xlsWorkBook{*wb, f}, nil
	case ".xlsx":
		wb, err := xlsx.OpenReaderAt(f, info.Size())
		if err != nil {
			return nil, err
		}
		return &xlsxWorkBook{*wb, f}, nil
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

func indexOfString(row []string, name string) int {
	for i, cell := range row {
		if cell == name {
			return i
		}
	}
	return -1
}
