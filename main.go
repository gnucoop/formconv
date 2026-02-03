package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gnucoop/formconv/formats"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Fprintln(os.Stderr, `No input files provided.
formconv converts xlsform files to ajf. Usage:
formconv form1.xlsx form2.xls`)
		return
	}

	for _, fileName := range os.Args[1:] {
		var err error
		ext := filepath.Ext(fileName)
		switch ext {
		case ".xls", ".xlsx":
			err = decXlsEncAjf(fileName)
		case ".json":
			err = decAjfEncXls(fileName)
		default:
			err = fmt.Errorf("Unrecognized file format %s", ext)
		}
		if err != nil {
			log.Fatal(fmt.Errorf("%s: %s", fileName, err))
		}
	}
}

func decXlsEncAjf(xlsName string) error {
	f, err := os.Open(xlsName)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}
	ext := filepath.Ext(xlsName)
	wb, err := formats.NewWorkBook(f, ext, stat.Size())
	if err != nil {
		return fmt.Errorf("Error opening workbook: %s", err)
	}
	xls, err := formats.DecXlsform(wb)
	if err != nil {
		return fmt.Errorf("Error decoding xlsform: %s", err)
	}
	ajf, err := formats.Convert(xls)
	if err != nil {
		return err
	}
	name := xlsName[0 : len(xlsName)-len(ext)]
	ajfName := name + ".json"
	err = formats.EncJsonToFile(ajfName, ajf)
	if err != nil {
		return fmt.Errorf("Error encoding json: %s", err)
	}
	return nil
}

func decAjfEncXls(ajfName string) error {
	f, err := os.Open(ajfName)
	if err != nil {
		return err
	}
	defer f.Close()

	var ajf formats.AjfForm
	err = formats.DecJson(f, &ajf)
	if err != nil {
		return fmt.Errorf("Error decoding ajf form: %s", err)
	}
	xlsform, err := formats.Revert(&ajf)
	if err != nil {
		return err
	}
	excel := formats.XlsFormToExcel(xlsform)
	ext := filepath.Ext(ajfName)
	name := ajfName[0 : len(ajfName)-len(ext)]
	excelName := name + ".xlsx"
	err = excel.Save(excelName)
	if err != nil {
		return fmt.Errorf("Error saving file: %s", err)
	}
	return nil
}
