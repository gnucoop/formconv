package formats

import (
	"fmt"

	"github.com/tealeg/xlsx"
)

func Revert(ajf *AjfForm) (*XlsForm, error) {
	var xls XlsForm

	for _, origin := range ajf.ChoicesOrigins {
		for _, choice := range origin.Choices {
			value := choice["value"]
			label := choice["label"]
			choicesRow := MakeChoicesRow("list name", origin.Name, "name", value, "label", label)
			xls.Choices = append(xls.Choices, choicesRow)
		}
	}

	surveyRows, err := nodesToSurvey(ajf.Slides)
	if err != nil {
		return nil, err
	}
	xls.Survey = surveyRows

	for _, tag := range ajf.StringIdentifier {
		if tag.Label != "" && tag.Value[0] != "" {
			settingsRow := MakeSettingsRow("tag label", tag.Label, "tag value", tag.Value[0])
			xls.Settings = append(xls.Settings, settingsRow)
		}
	}
	return &xls, nil
}

func nodesToSurvey(nodes []Node) ([]SurveyRow, error) {
	var surveyRows []SurveyRow

	for _, node := range nodes {
		if node.Type == NtField {
			fieldRow, err := fieldToSurveyRow(node)
			if err != nil {
				return nil, err
			}
			surveyRows = append(surveyRows, fieldRow)
			continue
		}
		// Other node types should be groups, slides and repeating slides
		rowType := "begin group"
		if node.Type == NtRepeatingSlide {
			rowType = "begin repeat"
		}
		beginRow := MakeSurveyRow("type", rowType, "name", node.Name, "label", node.Label)
		if node.Type == NtRepeatingSlide && node.MaxReps != nil {
			beginRow.cells["repeat_count"] = fmt.Sprintf("%d", *node.MaxReps)
		}
		if node.Visibility != nil && node.Visibility.Condition != "" {
			beginRow.cells["relevant"] = "js: " + node.Visibility.Condition
		}
		if node.ReadOnly != nil && node.ReadOnly.Condition != "" {
			beginRow.cells["readonly"] = "js: " + node.ReadOnly.Condition
		}
		surveyRows = append(surveyRows, beginRow)

		childRows, err := nodesToSurvey(node.Nodes)
		if err != nil {
			return nil, err
		}
		surveyRows = append(surveyRows, childRows...)

		endRowType := "end group"
		if node.Type == NtRepeatingSlide {
			endRowType = "end repeat"
		}
		endRow := MakeSurveyRow("type", endRowType)
		surveyRows = append(surveyRows, endRow)
	}
	return surveyRows, nil
}

func fieldToSurveyRow(node Node) (SurveyRow, error) {
	if node.FieldType == nil {
		return SurveyRow{}, fmt.Errorf("Field %s has no field type", node.Name)
	}
	row := MakeSurveyRow("type", rowType(node), "name", node.Name, "label", node.Label, "hint", node.Hint)

	switch *node.FieldType {
	case FtSingleChoice, FtMultipleChoice:
		if node.ForceNarrow {
			row.cells["appearance"] = "minimal"
		}
	case FtRange:
		start, end, step := 0, 10, 1
		if node.RangeStart != nil {
			start = *node.RangeStart
		}
		if node.RangeEnd != nil {
			end = *node.RangeEnd
		}
		if node.RangeStep != nil {
			step = *node.RangeStep
		}
		row.cells["parameters"] = fmt.Sprintf("start=%d end=%d step=%d", start, end, step)
		row.cells["appearance"] = node.Appearance
	case FtText:
		row.cells["appearance"] = "multiline"
	case FtNote:
		row.cells["label"] = node.HTML
	case FtFormula:
		if node.Formula != nil && node.Formula.Formula != "" {
			row.cells["calculation"] = "js: " + node.Formula.Formula
		}
	case FtSignature:
		row.cells["appearance"] = "signature"
	}

	if node.Visibility != nil && node.Visibility.Condition != "" {
		row.cells["relevant"] = "js: " + node.Visibility.Condition
	}
	if node.Editable != nil && !*node.Editable {
		row.cells["readonly"] = "yes"
	}
	if node.Validation != nil {
		if node.Validation.NotEmpty {
			row.cells["required"] = "yes"
			row.cells["required_message"] = node.Validation.NotEmptyMsg
		}
		if len(node.Validation.Conditions) > 0 {
			cond := node.Validation.Conditions[0]
			row.cells["constraint"] = "js: " + cond.Condition
			row.cells["constraint_message"] = cond.ErrorMessage
		}
	}
	if node.DefaultVal != nil && node.DefaultVal.Formula != "" {
		row.cells["default"] = "js: " + node.DefaultVal.Formula
	}
	return row, nil
}

func rowType(node Node) string {
	switch *node.FieldType {
	case FtString, FtText:
		return "text"
	case FtNumber:
		return "decimal"
	case FtBoolean:
		return "boolean"
	case FtSingleChoice:
		return "select_one " + node.ChoicesOriginRef
	case FtMultipleChoice:
		return "select_multiple " + node.ChoicesOriginRef
	case FtFormula:
		return "calculate"
	case FtNote:
		return "note"
	case FtDate:
		return "date"
	case FtTime:
		return "time"
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
		return "image"
	case FtAudio:
		return "audio"
	default:
		return "text"
	}
}

func XlsFormToExcel(xls *XlsForm) *xlsx.File {
	file := xlsx.NewFile()

	surveySheet, _ := file.AddSheet("survey")
	surveyHeaders := []string{"type", "name", "label", "hint", "required", "required_message",
		"relevant", "readonly", "default", "constraint", "constraint_message",
		"calculation", "appearance", "parameters", "repeat_count"}
	currentRow := surveySheet.AddRow()
	for _, header := range surveyHeaders {
		currentRow.AddCell().Value = header
	}
	for _, row := range xls.Survey {
		currentRow = surveySheet.AddRow()
		currentRow.AddCell().Value = row.Type
		currentRow.AddCell().Value = row.Name()
		currentRow.AddCell().Value = row.Label("")
		currentRow.AddCell().Value = row.Hint("")
		currentRow.AddCell().Value = row.Required()
		currentRow.AddCell().Value = row.RequiredMessage("")
		currentRow.AddCell().Value = row.Relevant()
		currentRow.AddCell().Value = row.ReadOnly()
		currentRow.AddCell().Value = row.Default()
		currentRow.AddCell().Value = row.Constraint()
		currentRow.AddCell().Value = row.ConstraintMsg("")
		currentRow.AddCell().Value = row.Calculation()
		currentRow.AddCell().Value = row.Appearance()
		currentRow.AddCell().Value = row.Parameters()
		currentRow.AddCell().Value = row.RepeatCount()
	}

	choicesSheet, _ := file.AddSheet("choices")
	choicesHeaders := []string{"list name", "name", "label"}
	currentRow = choicesSheet.AddRow()
	for _, header := range choicesHeaders {
		currentRow.AddCell().Value = header
	}
	for _, row := range xls.Choices {
		currentRow = choicesSheet.AddRow()
		currentRow.AddCell().Value = row.ListName()
		currentRow.AddCell().Value = row.Name()
		currentRow.AddCell().Value = row.Label("")
	}

	if len(xls.Settings) > 0 {
		settingsSheet, _ := file.AddSheet("settings")
		settingsHeaders := []string{"tag label", "tag value"}
		currentRow = settingsSheet.AddRow()
		for _, header := range settingsHeaders {
			currentRow.AddCell().Value = header
		}
		for _, row := range xls.Settings {
			currentRow = settingsSheet.AddRow()
			currentRow.AddCell().Value = row.TagLabel()
			currentRow.AddCell().Value = row.TagValue()
		}
	}
	return file
}
