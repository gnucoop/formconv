package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"bitbucket.org/gnucoop/xls2ajf/formats"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set!")
	}

	http.Handle("/", http.FileServer(http.Dir("server/static")))
	http.HandleFunc("/result.json", convert)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func convert(w http.ResponseWriter, r *http.Request) {
	f, head, err := r.FormFile("excelFile")
	if err != nil {
		log.Println(err)
		return
	}
	xls, err := formats.DecXls(f, filepath.Ext(head.Filename), head.Size)
	if err != nil {
		log.Println(err)
		return
	}
	ajf, err := formats.Xls2ajf(xls)
	if err != nil {
		log.Println(err)
		return
	}
	err = formats.EncAjf(w, ajf)
	if err != nil {
		log.Println(err)
		return
	}
}
