package formats

import (
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/360EntSecGroup-Skylar/excelize"
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
	for i, sheet := range sheetInfos {
		if !sheet.mandatory && !f.HasSheet(sheet.name) {
			continue
		}
		rows, err := f.GetRows(sheet.name)
		if err != nil {
			return nil, fmt.Errorf("Couldn't get sheet %q from file %s: %s", sheet.name, fileName, err)
		}
		rows = deleteEmpty(rows)
		if len(rows) == 0 {
			return nil, fmt.Errorf("Empty sheet %q in file %s", sheet.name, fileName)
		}
		head := rows[0]
		rows = rows[1:]
		colIndices := make([]int, len(sheet.columns))
		for i, colInfo := range sheet.columns {
			colIndices[i] = indexOfString(head, colInfo.name)
			if colInfo.mandatory && colIndices[i] == -1 {
				return nil, fmt.Errorf("Error in file %s, sheet %q: column %q is mandatory",
					fileName, sheet.name, colInfo.name)
			}
		}
		sheetSlice := formVal.Field(i)
		for _, row := range rows {
			rowVal := reflect.New(sheetSlice.Type().Elem()).Elem()
			for j := range sheet.columns {
				if colIndices[j] != -1 {
					rowVal.Field(j).Set(reflect.ValueOf(row[colIndices[j]]))
				}
			}
			sheetSlice.Set(reflect.Append(sheetSlice, rowVal))
		}
	}
	return &form, nil
}

type excelFile interface {
	HasSheet(string) bool
	GetRows(sheet string) ([][]string, error)
	Close() error
}

type xlsxFile excelize.File

func (f *xlsxFile) HasSheet(sheet string) bool {
	return (*excelize.File)(f).GetSheetIndex(sheet) != 0
}
func (f *xlsxFile) GetRows(sheet string) ([][]string, error) {
	return (*excelize.File)(f).GetRows(sheet)
}
func (f *xlsxFile) Close() error {
	return fmt.Errorf("Closing files is not supported by excelize")
}

// Support for old .xls files can be added here.

func openExcelFile(name string) (excelFile, error) {
	ext := filepath.Ext(name)
	if ext != ".xlsx" {
		return nil, fmt.Errorf("Unsupported excel file type: %s", ext)
	}
	f, err := excelize.OpenFile(name)
	return (*xlsxFile)(f), err
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
