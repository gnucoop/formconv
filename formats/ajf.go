package formats

import (
	"bufio"
	"encoding/json"
	"fmt"
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
	Validation       *FieldValidation `json:"validation,omitempty"`
	Nodes            []Node           `json:"nodes,omitempty"`
}

type NodeType int

const (
	NtField NodeType = iota
	_
	NtGroup
	NtSlide
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
)

type FieldValidation struct {
	NotEmpty bool `json:"notEmpty,omitempty"`
}

func EncAjfToFile(form *AjfForm, fileName string) (err error) {
	var f *os.File
	f, err = os.Create(fileName)
	if err != nil {
		return fmt.Errorf("Could not create file %s: %s", fileName, err)
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(fileName)
		}
	}()

	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	err = enc.Encode(form)
	if err != nil {
		return fmt.Errorf("Could not encode ajf form: %s", err)
	}
	err = w.Flush()
	if err != nil {
		return fmt.Errorf("Error flushing form to file %s: %s", fileName, err)
	}
	return nil
}
