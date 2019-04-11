package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) <= 1 {
		log.Println(`No input files provided.
xls2ajf converts xlsform files to ajf. Usage:
xls2ajf form1.xlsx form2.xlsx`)
		return
	}

	for _, fileName := range os.Args[1:] {
		err := decXlsEncAjf(fileName)
		if err != nil {
			log.Println(err)
		}
	}
}

func decXlsEncAjf(xlsName string) error {
	xls, err := DecXlsFromFile(xlsName)
	if err != nil {
		return err
	}
	ajf, err := Xls2ajf(xls)
	if err != nil {
		return fmt.Errorf("Error converting file %s: %s", xlsName, err)
	}
	ext := filepath.Ext(xlsName)
	ajfName := xlsName[0:len(xlsName)-len(ext)] + ".json"
	err = EncAjfToFile(ajf, ajfName)
	if err != nil {
		return err
	}
	return nil
}
