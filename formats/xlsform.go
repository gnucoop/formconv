package formats

import (
	"fmt"
	"reflect"

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
	f, closer, err := xls.OpenWithCloser(fileName, "utf-8")
	if err != nil {
		return nil, fmt.Errorf("Could not open excel file %s: %s", fileName, err)
	}
	defer closer.Close()

	var form XlsForm
	formVal := reflect.ValueOf(&form).Elem()
	for i, sheetInfo := range sheetInfos {
		sheet := getSheet(f, sheetInfo.name)
		if sheet == nil && sheetInfo.mandatory {
			return nil, fmt.Errorf("Missing mandatory sheet %q in file %s", sheetInfo.name, fileName)
		}
		if sheet == nil {
			continue // not mandatory, skip
		}
		headIndex, head := getFirstRow(sheet)
		if head == nil {
			return nil, fmt.Errorf("Empty sheet %q in file %s", sheetInfo.name, fileName)
		}
		colIndices := make([]int, len(sheetInfo.columns))
		for i, colInfo := range sheetInfo.columns {
			colIndices[i] = columnIndex(head, colInfo.name)
			if colIndices[i] == -1 && colInfo.mandatory {
				return nil, fmt.Errorf("Error in file %s, sheet %q: column %q is mandatory",
					fileName, sheetInfo.name, colInfo.name)
			}
		}
		destSlice := formVal.Field(i)
		for i := headIndex + 1; i <= int(sheet.MaxRow); i++ {
			row := sheet.Row(i)
			if row == nil {
				continue // empty rows come out as nil
			}
			destRow := reflect.New(destSlice.Type().Elem()).Elem()
			for j := range sheetInfo.columns {
				if colIndices[j] != -1 {
					destRow.Field(j).Set(reflect.ValueOf(row.Col(colIndices[j])))
				}
			}
			destSlice.Set(reflect.Append(destSlice, destRow))
		}
	}
	return &form, nil
}

func getSheet(w *xls.WorkBook, sheet string) *xls.WorkSheet {
	for i := 0; i < w.NumSheets(); i++ {
		if s := w.GetSheet(i); s.Name == sheet {
			return s
		}
	}
	return nil
}

func getFirstRow(sheet *xls.WorkSheet) (int, *xls.Row) {
	for i := 0; i <= int(sheet.MaxRow); i++ {
		if row := sheet.Row(i); row != nil {
			return i, row
		}
	}
	return -1, nil
}

func columnIndex(row *xls.Row, cell string) int {
	for i := row.FirstCol(); i <= row.LastCol(); i++ {
		if row.Col(i) == cell {
			return i
		}
	}
	return -1
}
