package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gnucoop/formconv/formats"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set!")
	}

	http.Handle("/", http.FileServer(http.Dir("./server/static")))
	http.HandleFunc("/result.json", convert)
	http.HandleFunc("/result.xlsx", revert)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func setAllowOrigins(h http.Header) { h.Set("Access-Control-Allow-Origin", "*") }

func convert(w http.ResponseWriter, r *http.Request) {
	setAllowOrigins(w.Header())

	switch r.Method {
	case http.MethodOptions:
		// OK
	case http.MethodGet:
		fmt.Fprintln(w, "You should POST an excel file here.")
	case http.MethodPost:
		convertPost(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Unsupported method %s", r.Method)
	}
}

func revert(w http.ResponseWriter, r *http.Request) {
	setAllowOrigins(w.Header())

	switch r.Method {
	case http.MethodOptions:
		// OK
	case http.MethodGet:
		fmt.Fprintln(w, "You should POST a JSON file here.")
	case http.MethodPost:
		revertPost(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Unsupported method %s", r.Method)
	}
}

func convertPost(w http.ResponseWriter, r *http.Request) {
	f, head, err := r.FormFile("excelFile")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error retrieving POST file: %s", err)
		return
	}
	defer f.Close()

	wb, err := formats.NewWorkBook(f, filepath.Ext(head.Filename), head.Size)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintf(w, "Error opening workbook: %s", err)
		return
	}
	xls, err := formats.DecXlsform(wb)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintf(w, "Error decoding xlsform: %s", err)
		return
	}
	ajf, err := formats.Convert(xls)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintln(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err = formats.EncIndentedJson(w, ajf)
	if err != nil {
		log.Printf("Error writing json response: %s", err)
	}
}

func revertPost(w http.ResponseWriter, r *http.Request) {
	f, _, err := r.FormFile("jsonFile")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error retrieving POST file: %s", err)
		return
	}
	defer f.Close()

	var ajf formats.AjfForm
	err = formats.DecJson(f, &ajf)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintf(w, "Error decoding JSON: %s", err)
		return
	}
	xlsform, err := formats.Revert(&ajf)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintln(w, err)
		return
	}
	excel := formats.XlsFormToExcel(xlsform)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=result.xlsx")
	err = excel.Write(w)
	if err != nil {
		log.Printf("Error writing excel response: %s", err)
	}
}
