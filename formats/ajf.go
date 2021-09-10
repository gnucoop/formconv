package formats

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

type AjfForm struct {
	StringIdentifier []Tag           `json:"stringIdentifier,omitempty"`
	ChoicesOrigins   []ChoicesOrigin `json:"choicesOrigins,omitempty"`
	Slides           []Node          `json:"nodes"`
}

type Tag struct {
	Label string    `json:"label"`
	Value [1]string `json:"value"`
}

type ChoicesOrigin struct {
	Type        OriginType `json:"type"`
	Name        string     `json:"name"`
	ChoicesType ChoiceType `json:"choicesType"`
	Choices     []Choice   `json:"choices"`
}

type OriginType string

const OtFixed OriginType = "fixed"

type ChoiceType string

const CtString ChoiceType = "string"

// Choice has fields "value", "label" and possibly others
// defined by the user to be used in choice filters.
type Choice map[string]string

type Node struct {
	Previous int      `json:"parent"`
	Id       int      `json:"id"`
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Hint     string   `json:"hint,omitempty"`
	Editable *bool    `json:"editable,omitempty"`
	Type     NodeType `json:"nodeType"`

	FieldType        *FieldType       `json:"fieldType,omitempty"`
	RangeStart       *int             `json:"start,omitempty"`
	RangeEnd         *int             `json:"end,omitempty"`
	RangeStep        *int             `json:"step,omitempty"`
	ChoicesOriginRef string           `json:"choicesOriginRef,omitempty"`
	ChoicesFilter    *Formula         `json:"choicesFilter,omitempty"`
	HTML             string           `json:"HTML,omitempty"`
	MaxReps          *int             `json:"maxReps,omitempty"`
	Formula          *Formula         `json:"formula,omitempty"`
	Validation       *FieldValidation `json:"validation,omitempty"`
	Visibility       *NodeVisibility  `json:"visibility,omitempty"`
	Nodes            []Node           `json:"nodes,omitempty"`
}

type NodeType int

const (
	NtField NodeType = iota
	_
	NtGroup
	NtSlide
	NtRepeatingSlide
)

type FieldType int

var (
	FtString         FieldType = 0
	FtText           FieldType = 1
	FtNumber         FieldType = 2
	FtBoolean        FieldType = 3
	FtSingleChoice   FieldType = 4
	FtMultipleChoice FieldType = 5
	FtFormula        FieldType = 6
	FtNote           FieldType = 7
	FtDate           FieldType = 9
	FtTime           FieldType = 10
	FtGeolocation    FieldType = 12
	FtBarcode        FieldType = 13
	FtFile           FieldType = 14
	FtImage          FieldType = 15
	FtVideoUrl       FieldType = 16
	FtRange          FieldType = 17
)

type Formula struct {
	Formula string `json:"formula"`
}

type FieldValidation struct {
	NotEmpty   bool                  `json:"notEmpty,omitempty"`
	Conditions []ValidationCondition `json:"conditions,omitempty"`
}

type ValidationCondition struct {
	Condition        string `json:"condition"`
	ClientValidation bool   `json:"clientValidation"` // always true
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

type NodeVisibility struct {
	Condition string `json:"condition"`
}

func EncIndentedJson(w io.Writer, e interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	enc.SetEscapeHTML(false)
	return enc.Encode(e)
}

func EncJsonToFile(fileName string, e interface{}) (err error) {
	var f *os.File
	f, err = os.Create(fileName)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(fileName)
		}
	}()

	w := bufio.NewWriter(f)
	err = EncIndentedJson(w, e)
	if err != nil {
		return err
	}
	err = w.Flush()
	if err != nil {
		return err
	}
	return nil
}
