package rfortune

import (
	"fmt"
	"net/http"
)

var addr = "0.0.0.0:8080"

var foo, _ = fmt.Printf("")

func Start() {
	logger.Printf("Starting http Server on http://%s", addr)

	http.Handle("/", http.HandlerFunc(randomFortune))

	err := http.ListenAndServe(addr, nil)
	checkErr(err, "ListenAndServe Error :")
}

func randomFortune(w http.ResponseWriter, req *http.Request) {
	f, err := RandomFortune("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write([]byte(f.AsHtml()))
	}
}
