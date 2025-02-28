package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	serveMux := http.NewServeMux()
	s := &http.Server{
		Addr:           ":8080",
		Handler:        serveMux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
