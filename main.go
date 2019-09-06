package main

import (
	"fmt"
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
		err := decXlsEncAjf(fileName)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
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
	wb, err := formats.NewWorkBook(f, filepath.Ext(xlsName), stat.Size())
	if err != nil {
		return fmt.Errorf("Error opening workbook: %s", err)
	}
	xls, err := formats.DecXlsform(wb)
	if err != nil {
		return fmt.Errorf("Error decoding file %s: %s", xlsName, err)
	}
	ajf, err := formats.Convert(xls)
	if err != nil {
		return fmt.Errorf("%s, %s", xlsName, err)
	}
	ext := filepath.Ext(xlsName)
	name := xlsName[0 : len(xlsName)-len(ext)]
	ajfName := name + ".json"
	err = formats.EncJsonToFile(ajfName, ajf)
	if err != nil {
		return fmt.Errorf("Error encoding file %s: %s", ajfName, err)
	}

	// Translation files in case of multiple languages:
	survey := wb.Rows("survey")
	langs := formats.ListLanguages(survey)
	if len(langs) == 0 {
		return nil
	}
	choices := wb.Rows("choices")
	for lang := range langs {
		surveyTr := formats.Translation(survey, lang)
		choicesTr := formats.Translation(choices, lang)
		tr := formats.MergeMaps(surveyTr, choicesTr)
		err := formats.EncJsonToFile(name+"_"+lang+".json", tr)
		if err != nil {
			return err
		}
	}
	return nil
}
