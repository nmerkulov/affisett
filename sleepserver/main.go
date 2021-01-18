package main

import (
	"log"
	"net/http"
	"path"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		d, err := time.ParseDuration(path.Base(r.URL.Path))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		time.Sleep(d)
		w.Write([]byte("ok"))
	})

	log.Fatal(http.ListenAndServe(":8081", nil))
}
