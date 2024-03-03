package main

import (
	"log"
	"net/http"
	"toolkit"
)

func downloadFile(w http.ResponseWriter, r *http.Request) {
	var t toolkit.Tools

	t.DownloadStaticFile(w, r, "./files", "img.png", "downloaded-image.png")
}

func routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/download", downloadFile)

	return mux
}

func main() {
	mux := routes()

	err := http.ListenAndServe(":3000", mux)
	if err != nil {
		log.Fatal(err)
	}
}
