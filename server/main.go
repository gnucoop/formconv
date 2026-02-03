package main

import (
	"fmt"
	"io/ioutil"
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
	http.HandleFunc("/xresult.json", convertJson)

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

func convertJson(w http.ResponseWriter, r *http.Request) {
	setAllowOrigins(w.Header())

	switch r.Method {
	case http.MethodOptions:
		// OK
	case http.MethodGet:
		fmt.Fprintln(w, "You should POST a JSON file here.")
	case http.MethodPost:
		convertJsonPost(w, r)
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

func convertJsonPost(w http.ResponseWriter, r *http.Request) {
	// Read JSON from request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error reading request body: %s", err)
		return
	}

	// Decode AJF form
	var ajf formats.AjfForm
	err = formats.DecodeJson(body, &ajf)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error decoding JSON: %s", err)
		return
	}

	// Convert AJF to XLS form
	xls, err := formats.Revert(&ajf)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintf(w, "Error converting JSON to XLS form: %s", err)
		return
	}

	// Convert XLS form to Excel file
	excelFile, err := formats.ConvertXlsFormToExcel(xls)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error creating Excel file: %s", err)
		return
	}

	// Save Excel file to temporary location and send it back
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=converted_form.xlsx")
	err = excelFile.Write(w)
	if err != nil {
		log.Printf("Error writing excel response: %s", err)
	}
}
