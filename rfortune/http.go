package rfortune

import (
	"fmt"
	"net/http"
)

var addr = "0.0.0.0:8080"

func Start() {
	logger.Printf("Starting http Server on http://%s", addr)

	http.Handle("/", http.HandlerFunc(randomFortune))

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Printf("ListenAndServe Error :", err)
	}
}

func randomFortune(w http.ResponseWriter, req *http.Request) {
	f := RandomFortune("")
	w.Write([]byte(f.AsHtml()))
}
