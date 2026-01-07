package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gnucoop/formconv/formats"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Fprintln(os.Stderr, `No input files provided.

xformconv converts AJF JSON files back to XLSX format. Usage:
xformconv form1.json form2.json`)
		return
	}

	for _, fileName := range os.Args[1:] {
		err := convertJsonToXlsx(fileName)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func convertJsonToXlsx(jsonName string) error {
	// Read JSON file
	jsonData, err := ioutil.ReadFile(jsonName)
	if err != nil {
		return fmt.Errorf("Error reading JSON file %s: %s", jsonName, err)
	}

	// Decode AJF form
	var ajf formats.AjfForm
	err = formats.DecodeJson(jsonData, &ajf)
	if err != nil {
		return fmt.Errorf("Error decoding JSON %s: %s", jsonName, err)
	}

	// Convert AJF to XLS form
	xls, err := formats.ConvertAjfToXlsform(&ajf)
	if err != nil {
		return fmt.Errorf("Error converting JSON to XLS form %s: %s", jsonName, err)
	}

	// Convert XLS form to Excel file
	excelFile, err := formats.ConvertXlsFormToExcel(xls)
	if err != nil {
		return fmt.Errorf("Error creating Excel file %s: %s", jsonName, err)
	}

	// Save Excel file
	ext := filepath.Ext(jsonName)
	name := jsonName[0 : len(jsonName)-len(ext)]
	excelName := name + ".xlsx"
	err = excelFile.Save(excelName)
	if err != nil {
		return fmt.Errorf("Error saving Excel file %s: %s", excelName, err)
	}

	fmt.Printf("Successfully converted %s to %s\n", jsonName, excelName)
	return nil
}