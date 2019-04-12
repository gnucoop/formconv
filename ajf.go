package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type AjfForm struct {
	ChoicesOrigins []ChoicesOrigin `json:"choicesOrigins,omitempty"`
	Slides         []Slide         `json:"nodes"`
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

type Slide struct {
	Id       int      `json:"id"`
	Parent   int      `json:"parent"`
	NodeType NodeType `json:"nodeType"` // always ntSlide
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Fields   []Field  `json:"nodes"`
}

type Field struct {
	Id               int              `json:"id"`
	Parent           int              `json:"parent"`
	NodeType         NodeType         `json:"nodeType"` // always ntField
	FieldType        FieldType        `json:"fieldType"`
	Name             string           `json:"name"`
	Label            string           `json:"label"`
	ChoicesOriginRef string           `json:"choicesOriginRef,omitempty"`
	HTML             string           `json:"HTML,omitempty"`
	Validation       *FieldValidation `json:"validation,omitempty"`
}

type NodeType int

const (
	NtField NodeType = 0
	NtSlide NodeType = 3
)

type FieldType int

const (
	FtString FieldType = iota
	FtText
	FtNumber
	FtBoolean
	FtSingleChoice
	FtMultipleChoice
	FtFormula
	FtEmpty
	FtDate
	FtDateInput
	//FtTime
	//FtTable
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
