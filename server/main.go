package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	log.Println("pinkgopher server starting.")
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set!")
	}

	http.Handle("/", http.FileServer(http.Dir("server/static")))
	http.HandleFunc("/upload", upload)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func upload(w http.ResponseWriter, r *http.Request) {
	_, head, err := r.FormFile("uploadfile")
	if err != nil {
		log.Println(err)
		return
	}
	resp := fmt.Sprintf("Thanks for uploading your file! What a nice file header:\n%v\n", head)
	_, err = io.WriteString(w, resp)
	if err != nil {
		log.Println(err)
	}
}
