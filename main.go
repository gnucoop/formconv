package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zan8rob/formconv/formats"
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
	_, xlsShort := filepath.Split(xlsName)
	xls, err := formats.DecXlsFromFile(xlsName)
	if err != nil {
		return fmt.Errorf("Error decoding file %s: %s", xlsShort, err)
	}
	ajf, err := formats.Convert(xls)
	if err != nil {
		return fmt.Errorf("Error converting file %s: %s", xlsShort, err)
	}
	ext := filepath.Ext(xlsName)
	ajfName := xlsName[0:len(xlsName)-len(ext)] + ".json"
	_, ajfShort := filepath.Split(ajfName)
	err = formats.EncAjfToFile(ajfName, ajf)
	if err != nil {
		return fmt.Errorf("Error encoding file %s: %s", ajfShort, err)
	}
	return nil
}
