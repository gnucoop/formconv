package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type ajfForm struct {
	ChoicesOrigins []choicesOrigin `json:"choicesOrigins,omitempty"`
	Slides         []slide         `json:"nodes"`
}

type choicesOrigin struct {
	Type        originType `json:"type"`
	Name        string     `json:"name"`
	ChoicesType choiceType `json:"choicesType"`
	Choices     []choice   `json:"choices"`
}

type originType string

const otFixed originType = "fixed"

type choiceType string

const ctString choiceType = "string"

type choice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type slide struct {
	Id       int      `json:"id"`
	Parent   int      `json:"parent"`
	NodeType nodeType `json:"nodeType"` // always ntSlide
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Fields   []field  `json:"nodes"`
}

type field struct {
	Id               int              `json:"id"`
	Parent           int              `json:"parent"`
	NodeType         nodeType         `json:"nodeType"` // always ntField
	FieldType        fieldType        `json:"fieldType"`
	Name             string           `json:"name"`
	Label            string           `json:"label"`
	ChoicesOriginRef string           `json:"choicesOriginRef,omitempty"`
	Validation       *fieldValidation `json:"validation,omitempty"`
}

type nodeType int

const (
	ntField nodeType = 0
	ntSlide nodeType = 3
)

type fieldType int

const (
	ftString fieldType = iota
	ftText
	ftNumber
	ftBoolean
	ftSingleChoice
	ftMultipleChoice
	ftFormula
	ftEmpty
	ftDate
	ftDateInput
	//ftTime
	//ftTable
)

type fieldValidation struct {
	NotEmpty bool `json:"notEmpty,omitempty"`
}

func encAjfToFile(form *ajfForm, fileName string) (err error) {
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
