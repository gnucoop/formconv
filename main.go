package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"bitbucket.org/gnucoop/xls2ajf/formats"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) <= 1 {
		log.Println(`No input files provided.
xls2ajf converts xlsform files to ajf. Usage:
xls2ajf form1.xls form2.xls`)
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
	_, xlsShort := filepath.Split(xlsName)
	xls, err := formats.DecXlsFromFile(xlsName)
	if err != nil {
		return fmt.Errorf("Error decoding file %s: %s", xlsShort, err)
	}
	ajf, err := formats.Xls2ajf(xls)
	if err != nil {
		return fmt.Errorf("Error converting file %s: %s", xlsShort, err)
	}
	ext := filepath.Ext(xlsName)
	ajfName := xlsName[0:len(xlsName)-len(ext)] + ".json"
	_, ajfShort := filepath.Split(ajfName)
	err = formats.EncAjfToFile(ajf, ajfName)
	if err != nil {
		return fmt.Errorf("Error encoding file %s: %s", ajfShort, err)
	}
	return nil
}
