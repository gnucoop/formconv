package main

import (
	"fmt"
	"path/filepath"

	"github.com/360EntSecGroup-Skylar/excelize"
)

type XlsForm struct {
	survey  surveySheet
	choices choicesSheet
}
type surveySheet struct {
	types, names, labels, required []string
	// relevant, constraint, default?, readonly? calculation?
}
type choicesSheet struct {
	listNames, names, labels []string
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

// Types used to define which sheets/columns to read from a file
// and where to store the columns.
type sheet struct {
	name      string
	columns   []column
	mandatory bool
}
type column struct {
	name      string
	dest      *[]string
	mandatory bool
}

func DecXlsFromFile(fileName string) (*XlsForm, error) {
	f, err := openExcelFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("Could not open excel file %s: %s", fileName, err)
	}
	defer f.Close()

	var form XlsForm
	sheets := []sheet{{
		name:      "survey",
		mandatory: true,
		columns: []column{
			{"type", &form.survey.types, true},
			{"name", &form.survey.names, true},
			{"label", &form.survey.labels, true},
			{"required", &form.survey.required, false},
		},
	}, {
		name:      "choices",
		mandatory: true,
		columns: []column{
			{"list_name", &form.choices.listNames, true},
			{"name", &form.choices.names, true},
			{"label", &form.choices.labels, true},
		},
	}}
	for _, sheet := range sheets {
		if !sheet.mandatory && !f.HasSheet(sheet.name) {
			continue
		}
		rows, err := f.GetRows(sheet.name)
		if err != nil {
			return nil, fmt.Errorf("Couldn't get sheet %q from file %s: %s",
				sheet.name, fileName, err)
		}
		rows = deleteEmpty(rows)
		cols := transpose(rows)
		cols = deleteEmpty(cols)
		for _, column := range sheet.columns {
			*column.dest = findCol(cols, column.name)
			if column.mandatory && *column.dest == nil {
				return nil, fmt.Errorf("Error in file %s, sheet %q: column %q is mandatory",
					fileName, sheet.name, column.name)
			}
		}
	}
	return &form, nil
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

func transpose(rows [][]string) [][]string {
	if len(rows) == 0 {
		return nil
	}
	cols := make([][]string, len(rows[0]))
	for i := range cols {
		cols[i] = make([]string, len(rows))
	}
	for i, row := range rows {
		for j, cell := range row {
			cols[j][i] = cell
		}
	}
	return cols
}

func findCol(cols [][]string, name string) []string {
	for _, col := range cols {
		if len(col) > 0 && col[0] == name {
			return col[1:]
		}
	}
	return nil
}
