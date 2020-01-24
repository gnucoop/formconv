package main

import (
	"encoding/json"
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
	http.HandleFunc("/translation.json", translate)

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

func translate(w http.ResponseWriter, r *http.Request) {
	setAllowOrigins(w.Header())

	switch r.Method {
	case http.MethodOptions:
		// OK
	case http.MethodGet:
		fmt.Fprintln(w, "You should POST an excel file here.")
	case http.MethodPost:
		translatePost(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Unsupported method %s", r.Method)
	}
}

func translatePost(w http.ResponseWriter, r *http.Request) {
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
		fmt.Fprintf(w, "Error opening excel workbook: %s", err)
		return
	}
	survey := wb.Rows("survey")
	choices := wb.Rows("choices")
	sourceLang := "English"
	if formats.HasDefaultLang(survey) {
		sourceLang = ""
	}
	targetLang := r.FormValue("lang")
	surveyTr := formats.Translation(survey, sourceLang, targetLang)
	choicesTr := formats.Translation(choices, sourceLang, targetLang)
	tr := formats.MergeMaps(surveyTr, choicesTr)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	enc.SetEscapeHTML(false)
	err = enc.Encode(tr)
	if err != nil {
		log.Printf("Error writing json response: %s", err)
	}
}
