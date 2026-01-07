package formats

import (
	"fmt"

	"github.com/tealeg/xlsx"
)

// ConvertAjfToXlsform converts AJF JSON schema back to XLSX format
func ConvertAjfToXlsform(ajf *AjfForm) (*XlsForm, error) {
	var xls XlsForm

	// Convert choices origins to choices rows
	for _, choiceOrigin := range ajf.ChoicesOrigins {
		if choiceOrigin.Type != OtFixed {
			continue // Skip non-fixed choice origins for now
		}
		for _, choice := range choiceOrigin.Choices {
			value := choice["value"]
			label := choice["label"]
			choicesRow := MakeChoicesRow("list name", choiceOrigin.Name, "name", value, "label", label)
			xls.Choices = append(xls.Choices, choicesRow)
		}
	}

	// Convert nodes to survey rows
	surveyRows, err := convertNodesToSurvey(ajf.Slides, 0)
	if err != nil {
		return nil, err
	}
	xls.Survey = surveyRows

	// Convert settings
	for _, tag := range ajf.StringIdentifier {
		if tag.Label != "" && tag.Value[0] != "" {
			settingsRow := MakeSettingsRow("tag label", tag.Label, "tag value", tag.Value[0])
			xls.Settings = append(xls.Settings, settingsRow)
		}
	}

	return &xls, nil
}

func convertNodesToSurvey(nodes []Node, parentID int) ([]SurveyRow, error) {
	var surveyRows []SurveyRow

	for _, node := range nodes {
		// Handle groups/repeats
		if node.Type == 3 || node.Type == 4 {
			// Add begin group/repeat
			rowType := "begin group"
			if node.Type == 4 {
				rowType = "begin repeat"
			}
			beginRow := MakeSurveyRow("type", rowType, "name", node.Name)
			if node.Label != "" {
				beginRow.cells["label"] = node.Label
			}
			surveyRows = append(surveyRows, beginRow)

			// Convert child nodes
			childRows, err := convertNodesToSurvey(node.Nodes, node.Id)
			if err != nil {
				return nil, err
			}
			surveyRows = append(surveyRows, childRows...)

			// Add end group/repeat
			endRowType := "end group"
			if node.Type == 4 {
				endRowType = "end repeat"
			}
			endRow := MakeSurveyRow("type", endRowType)
			surveyRows = append(surveyRows, endRow)
			continue
		}

		// Handle regular fields
		if node.Type == 0 {
			fieldRow, err := convertFieldToSurveyRow(node)
			if err != nil {
				return nil, err
			}
			surveyRows = append(surveyRows, fieldRow)
		}
	}

	return surveyRows, nil
}

func convertFieldToSurveyRow(node Node) (SurveyRow, error) {
	if node.FieldType == nil {
		return SurveyRow{}, fmt.Errorf("Field %s has no field type", node.Name)
	}

	// Create base row
	row := MakeSurveyRow("type", "text", "name", node.Name)
	if node.Label != "" {
		row.cells["label"] = node.Label
	}
	if node.Hint != "" {
		row.cells["hint"] = node.Hint
	}

	// Set the correct field type based on FieldType
	switch *node.FieldType {
	case 4, 5: // FtSingleChoice, FtMultipleChoice
		prefix := "select_one "
		if *node.FieldType == 5 {
			prefix = "select_multiple "
		}
		if node.ChoicesOriginRef != "" {
			row.cells["type"] = prefix + node.ChoicesOriginRef
			row.Type = prefix + node.ChoicesOriginRef
		} else {
			row.cells["type"] = prefix
			row.Type = prefix
		}
		if node.ChoicesFilter != nil && node.ChoicesFilter.Formula != "" {
			row.cells["choice_filter"] = node.ChoicesFilter.Formula
		}
		if node.ForceNarrow {
			row.cells["appearance"] = "minimal"
		}

	case 2: // FtNumber
		row.cells["type"] = "decimal"
		row.Type = "decimal"

	case 17: // FtRange
		row.cells["type"] = "range"
		row.Type = "range"
		if node.RangeStart != nil && node.RangeEnd != nil && node.RangeStep != nil {
			params := fmt.Sprintf("start=%d end=%d step=%d", *node.RangeStart, *node.RangeEnd, *node.RangeStep)
			row.cells["parameters"] = params
		}

	case 1: // FtText
		row.cells["type"] = "text"
		row.Type = "text"
		row.cells["appearance"] = "multiline"

	case 7: // FtNote
		row.cells["type"] = "note"
		row.Type = "note"
		if node.HTML != "" {
			row.cells["label"] = node.HTML
		}

	case 6: // FtFormula
		row.cells["type"] = "calculate"
		row.Type = "calculate"
		if node.Formula != nil && node.Formula.Formula != "" {
			row.cells["calculation"] = node.Formula.Formula
		}

	case 9: // FtDate
		row.cells["type"] = "date"
		row.Type = "date"

	case 10: // FtTime
		row.cells["type"] = "time"
		row.Type = "time"

	case 11: // FtTable
		row.cells["type"] = "table"
		row.Type = "table"

	case 12: // FtGeolocation
		row.cells["type"] = "geopoint"
		row.Type = "geopoint"

	case 13: // FtBarcode
		row.cells["type"] = "barcode"
		row.Type = "barcode"

	case 14: // FtFile
		row.cells["type"] = "file"
		row.Type = "file"

	case 15: // FtImage
		row.cells["type"] = "image"
		row.Type = "image"
		if node.Hint == "signature" {
			row.cells["appearance"] = "signature"
		}

	case 16: // FtVideoUrl
		row.cells["type"] = "video"
		row.Type = "video"

	case 3: // FtBoolean
		row.cells["type"] = "boolean"
		row.Type = "boolean"

	case 0: // FtString
		row.cells["type"] = "text"
		row.Type = "text"
	}

	// Handle validation
	if node.Validation != nil {
		if node.Validation.NotEmpty {
			row.cells["required"] = "yes"
			if node.Validation.NotEmptyMsg != "" {
				row.cells["required_message"] = node.Validation.NotEmptyMsg
			}
		}
		if len(node.Validation.Conditions) > 0 {
			// Use the first condition as the constraint
			if len(node.Validation.Conditions) > 0 {
				cond := node.Validation.Conditions[0]
				row.cells["constraint"] = cond.Condition
				if cond.ErrorMessage != "" {
					row.cells["constraint_message"] = cond.ErrorMessage
				}
			}
		}
	}

	// Handle default values
	if node.DefaultVal != nil && node.DefaultVal.Formula != "" {
		row.cells["default"] = node.DefaultVal.Formula
	}

	// Handle readonly
	if node.ReadOnly != nil && node.ReadOnly.Condition != "" {
		row.cells["readonly"] = node.ReadOnly.Condition
	}

	// Handle visibility
	if node.Visibility != nil && node.Visibility.Condition != "" {
		row.cells["relevant"] = node.Visibility.Condition
	}

	return row, nil
}

func getFieldTypeString(fieldType FieldType) string {
	switch fieldType {
	case FtString:
		return "text"
	case FtText:
		return "text"
	case FtNumber:
		return "decimal" // Default to decimal for numbers
	case FtBoolean:
		return "boolean"
	case FtSingleChoice:
		return "select_one" // Will be completed with choice list name
	case FtMultipleChoice:
		return "select_multiple" // Will be completed with choice list name
	case FtFormula:
		return "calculate"
	case FtNote:
		return "note"
	case FtDate:
		return "date"
	case FtTime:
		return "time"
	case FtTable:
		return "table"
	case FtGeolocation:
		return "geopoint"
	case FtBarcode:
		return "barcode"
	case FtFile:
		return "file"
	case FtImage:
		return "image"
	case FtVideoUrl:
		return "video"
	case FtRange:
		return "range"
	case FtSignature:
		return "image" // Signature is a type of image
	default:
		return "text"
	}
}

// ConvertXlsFormToExcel converts XlsForm back to Excel format
func ConvertXlsFormToExcel(xls *XlsForm) (*xlsx.File, error) {
	file := xlsx.NewFile()

	// Create survey sheet
	surveySheet, err := file.AddSheet("survey")
	if err != nil {
		return nil, err
	}

	// Add survey headers
	surveyHeaders := []string{"type", "name", "label", "hint", "required", "required_message", 
		"relevant", "readonly", "default", "constraint", "constraint_message", 
		"calculation", "appearance", "choice_filter", "parameters", "repeat_count"}

	surveyRow := surveySheet.AddRow()
	for _, header := range surveyHeaders {
		cell := surveyRow.AddCell()
		cell.Value = header
	}

	// Add survey data
	for _, row := range xls.Survey {
		surveyDataRow := surveySheet.AddRow()
		surveyDataRow.AddCell().Value = row.Type
		surveyDataRow.AddCell().Value = row.Name()
		surveyDataRow.AddCell().Value = row.Label("")
		surveyDataRow.AddCell().Value = row.Hint("")
		
		// Add other fields
		if row.Required() != "" {
			surveyDataRow.AddCell().Value = row.Required()
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.RequiredMessage("") != "" {
			surveyDataRow.AddCell().Value = row.RequiredMessage("")
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.Relevant() != "" {
			surveyDataRow.AddCell().Value = row.Relevant()
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.ReadOnly() != "" {
			surveyDataRow.AddCell().Value = row.ReadOnly()
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.Default() != "" {
			surveyDataRow.AddCell().Value = row.Default()
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.Constraint() != "" {
			surveyDataRow.AddCell().Value = row.Constraint()
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.ConstraintMsg("") != "" {
			surveyDataRow.AddCell().Value = row.ConstraintMsg("")
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.Calculation() != "" {
			surveyDataRow.AddCell().Value = row.Calculation()
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.Appearance() != "" {
			surveyDataRow.AddCell().Value = row.Appearance()
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.ChoiceFilter() != "" {
			surveyDataRow.AddCell().Value = row.ChoiceFilter()
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.Parameters() != "" {
			surveyDataRow.AddCell().Value = row.Parameters()
		} else {
			surveyDataRow.AddCell() // empty
		}
		
		if row.RepeatCount() != "" {
			surveyDataRow.AddCell().Value = row.RepeatCount()
		} else {
			surveyDataRow.AddCell() // empty
		}
	}

	// Create choices sheet
	choicesSheet, err := file.AddSheet("choices")
	if err != nil {
		return nil, err
	}

	// Add choices headers
	choicesHeaders := []string{"list name", "name", "label"}
	choicesHeaderRow := choicesSheet.AddRow()
	for _, header := range choicesHeaders {
		cell := choicesHeaderRow.AddCell()
		cell.Value = header
	}

	// Add choices data
	for _, row := range xls.Choices {
		choicesDataRow := choicesSheet.AddRow()
		choicesDataRow.AddCell().Value = row.ListName()
		choicesDataRow.AddCell().Value = row.Name()
		choicesDataRow.AddCell().Value = row.Label("")
	}

	// Create settings sheet if there are settings
	if len(xls.Settings) > 0 {
		settingsSheet, err := file.AddSheet("settings")
		if err != nil {
			return nil, err
		}

		// Add settings headers
		settingsHeaders := []string{"tag label", "tag value"}
		settingsHeaderRow := settingsSheet.AddRow()
		for _, header := range settingsHeaders {
			cell := settingsHeaderRow.AddCell()
			cell.Value = header
		}

		// Add settings data
		for _, row := range xls.Settings {
			settingsDataRow := settingsSheet.AddRow()
			settingsDataRow.AddCell().Value = row.TagLabel()
			settingsDataRow.AddCell().Value = row.TagValue()
		}
	}

	return file, nil
}