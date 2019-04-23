package formats

import (
	"fmt"
	"io"
	"path/filepath"
	"reflect"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/extrame/xls"
)

type XlsForm struct {
	Survey  []SurveyRow
	Choices []ChoicesRow
}
type SurveyRow struct {
	Type, Name, Label,
	Relevant, Constraint, Calculation, Required, Default string
}
type ChoicesRow struct {
	ListName, Name, Label string
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
			{name: "default"},
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
	f, err := openExcelFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("Could not open excel file %s: %s", fileName, err)
	}
	defer f.Close()

	var form XlsForm
	formVal := reflect.ValueOf(&form).Elem()
	for i, sheetInfo := range sheetInfos {
		sheetIndex := f.IndexOfSheet(sheetInfo.name)
		if sheetIndex == -1 && sheetInfo.mandatory {
			return nil, fmt.Errorf("Missing mandatory sheet %q in file %s", sheetInfo.name, fileName)
		}
		if sheetIndex == -1 {
			continue // not mandatory, skip
		}
		rows, err := f.Rows(sheetIndex)
		if err != nil {
			return nil, fmt.Errorf("Couldn't get sheet %q from file %s: %s",
				sheetInfo.name, fileName, err)
		}
		rows = deleteEmpty(rows)
		if len(rows) == 0 {
			return nil, fmt.Errorf("Empty sheet %q in file %s", sheetInfo.name, fileName)
		}
		head := rows[0]
		rows = rows[1:]
		colIndices := make([]int, len(sheetInfo.columns))
		for j, colInfo := range sheetInfo.columns {
			colIndices[j] = indexOfString(head, colInfo.name)
			if colIndices[j] == -1 && colInfo.mandatory {
				return nil, fmt.Errorf("Error in file %s, sheet %q: column %q is mandatory",
					fileName, sheetInfo.name, colInfo.name)
			}
		}
		destSlice := formVal.Field(i)
		for _, row := range rows {
			destRow := reflect.New(destSlice.Type().Elem()).Elem()
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

type excelFile interface {
	IndexOfSheet(name string) int
	Rows(sheet int) ([][]string, error)
	Close() error
}

type xlsxFile struct{ excelize.File }

func (f *xlsxFile) IndexOfSheet(name string) int {
	if i := f.GetSheetIndex(name); i != 0 {
		return i
	}
	return -1
}
func (f *xlsxFile) Rows(sheet int) ([][]string, error) {
	name := f.GetSheetName(sheet)
	if name == "" {
		return nil, fmt.Errorf("Invalid sheet index: %d", sheet)
	}
	return f.GetRows(name)
}
func (f *xlsxFile) Close() error {
	return fmt.Errorf("Closing files is not supported by excelize")
}

type xlsFile struct {
	xls.WorkBook
	io.Closer
}

func (f *xlsFile) IndexOfSheet(name string) int {
	for i := 0; i < f.NumSheets(); i++ {
		if f.GetSheet(i).Name == name {
			return i
		}
	}
	return -1
}
func (f *xlsFile) Rows(sheet int) ([][]string, error) {
	s := f.GetSheet(sheet)
	if s == nil {
		return nil, fmt.Errorf("Invalid sheet index: %d", sheet)
	}
	rows := make([][]string, s.MaxRow+1)
	numCols := 0
	for i := range rows {
		if row := s.Row(i); row != nil && row.LastCol()+1 > numCols {
			numCols = row.LastCol() + 1
		}
	}
	for i := range rows {
		rows[i] = make([]string, numCols)
		row := s.Row(i)
		if row == nil {
			continue
		}
		for j := range rows[i] {
			rows[i][j] = row.Col(j)
		}
	}
	return rows, nil
}

func openExcelFile(name string) (excelFile, error) {
	switch ext := filepath.Ext(name); ext {
	case ".xls":
		w, c, err := xls.OpenWithCloser(name, "utf-8")
		if err != nil {
			return nil, err
		}
		return &xlsFile{*w, c}, nil
	case ".xlsx":
		f, err := excelize.OpenFile(name)
		if err != nil {
			return nil, err
		}
		return &xlsxFile{*f}, nil
	default:
		return nil, fmt.Errorf("Unsupported excel file type: %s", ext)
	}
}

func deleteEmpty(rows [][]string) [][]string {
	filteredRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		empty := true
		for _, cell := range row {
			if cell != "" {
				empty = false
				break
			}
		}
		if !empty {
			filteredRows = append(filteredRows, row)
		}
	}
	return filteredRows
}

func indexOfString(row []string, name string) int {
	for i, cell := range row {
		if cell == name {
			return i
		}
	}
	return -1
}
