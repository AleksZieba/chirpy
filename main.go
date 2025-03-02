package main

import (
	"log"
	"net/http"
	//"time"
)

/*func main() {
	handler := http.FileServer(http.Dir("."))
	serveMux := http.NewServeMux()
	serveMux.Handle("/assets/logo.png", handler)
	s := &http.Server{
		Addr:           ":8080",
		Handler:        serveMux,
	}
	log.Fatal(s.ListenAndServe())
}*/ 

//var handler = http.FileServer(http.Dir("."))
//var serveMux = http.NewServeMux()

func initServer() *http.Server {
	handler := http.FileServer(http.Dir("."))
	serveMux := http.NewServeMux()
	serveMux.Handle("/assets/logo.png", handler)
	return &http.Server{
		Addr:           ":8080",
		Handler:        serveMux,
	}
}

func main() {
	s := initServer()
	log.Fatal(s.ListenAndServe())
}