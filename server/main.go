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

	http.Handle("/", http.FileServer(http.Dir("server/static")))
	http.HandleFunc("/result.json", handleConversion)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleConversion(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodOptions:
		optionsConversion(w, r)
	case http.MethodGet:
		getConversion(w, r)
	case http.MethodPost:
		postConversion(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Unsupported method %s", r.Method)
	}
}

func setAllowOrigins(h http.Header) {
	h.Set("Access-Control-Allow-Origin", "*")
}

func optionsConversion(w http.ResponseWriter, r *http.Request) {
	setAllowOrigins(w.Header())
	w.WriteHeader(http.StatusOK)
}

func getConversion(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "You should POST an excel file here.")
}

func postConversion(w http.ResponseWriter, r *http.Request) {
	setAllowOrigins(w.Header())
	f, head, err := r.FormFile("excelFile")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error retrieving POST file: %s", err)
		return
	}
	defer f.Close()

	xls, err := formats.DecXls(f, filepath.Ext(head.Filename), head.Size)
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
	err = formats.EncAjf(w, ajf)
	if err != nil {
		log.Printf("Error writing json response: %s", err)
	}
}
