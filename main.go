package main

import (
	"log"
	"net/http"
	"fmt" 
	"sync/atomic" 
	"strconv"
)

func main() {
	s := initServer()
	log.Fatal(s.ListenAndServe())
} 

var apiCfg = apiConfig{}

func initServer() *http.Server {
	fileServer := http.FileServer(http.Dir("./"))
	serveMux := http.NewServeMux()
	
	//wrappedFileServer := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer))
	//serveMux.Handle("/app/", wrappedFileServer) 
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))
	serveMux.HandleFunc("GET /api/healthz", healthzHandler) 
	serveMux.HandleFunc("GET /admin/metrics", totalHitsHandler) 
	serveMux.HandleFunc("POST /admin/reset", resetHitsHandler)
	return &http.Server{
		Addr:           ":8080",
		Handler:        serveMux,
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8") 
	w.WriteHeader(http.StatusOK) 
	_, err := w.Write([]byte("OK")) 
	if err != nil {
		fmt.Printf("error writing response: %v\n", err) 
		return 
	} 
} 

type apiConfig struct {
	fileserverHits atomic.Int32
} 

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Add(1) 
	next.ServeHTTP(w, r)
	})
} 

func totalHitsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK) 
	num := strconv.Itoa(int(apiCfg.fileserverHits.Load()))
	//msg := fmt.Sprintf("Hits: %d", apiCfg.fileserverHits)
	_, err := w.Write([]byte(fmt.Sprintf(`<html>
		<body>
		  <h1>Welcome, Chirpy Admin</h1>
		  <p>Chirpy has been visited %s times!</p>
		</body>
	  </html>`, num)))
	if err != nil {
		fmt.Printf("error writing response: %v\n", err)
		return 
	}
} 

func resetHitsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK) 
	apiCfg = apiConfig{}
} 