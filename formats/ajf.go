package formats

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

type AjfForm struct {
	ChoicesOrigins []ChoicesOrigin `json:"choicesOrigins,omitempty"`
	Slides         []Node          `json:"nodes"`
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

type Choice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Node struct {
	Previous int      `json:"parent"`
	Id       int      `json:"id"`
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Type     NodeType `json:"nodeType"`

	FieldType        *FieldType       `json:"fieldType,omitempty"`
	ChoicesOriginRef string           `json:"choicesOriginRef,omitempty"`
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
