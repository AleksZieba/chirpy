package main

import (
	"log"
	"net/http"
	"fmt" 
	"sync/atomic" 
	"strconv" 
	"io" 
	"encoding/json" 
	//"time"
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
	serveMux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)
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

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
        Body string `json:"body"` 
	}
	
	type returnVals struct {
        // the key will be the name of struct field unless you give it an explicit JSON tag
        //CreatedAt time.Time `json:"created_at"`
        //ID int `json:"id"` 
		Valid bool `json:"valid"`
    } 

	respBody := returnVals{
		//CreatedAt: time.Now(),
		//ID: 123, 
		Valid: true,
	}

	params := parameters{}
	
	bytesBody, err := io.ReadAll(r.Body) 
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(fmt.Sprintf(`{"error": "something went wrong"}`))) 
			if err != nil {
				log.Fatal("failed to write response")
			}
			//w.WriteHeader(400)
			return
		}
	err = json.Unmarshal(bytesBody, &params)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(fmt.Sprintf(`{"error": "something went wrong"}`))) 
			if err != nil {
				log.Fatal("failed to write response")
			}
			//w.WriteHeader(400)
			return
		}
	if len(params.Body) > 140 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, err := w.Write([]byte(fmt.Sprintf(`{"error": "Chirp is too long"}`))) 
		if err != nil {
			log.Fatal("failed to write response")
		}
		
		return
	} else { 
		dat, err := json.Marshal(respBody)
		if err != nil {
				log.Printf("Error marshalling JSON: %s", err)
				w.WriteHeader(500)
				return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(dat)
		if err != nil {
			log.Fatal("failed to write response")
			return 
		}
	}
} 


    //decoder := json.NewDecoder(r.Body)
    
/*    err := decoder.Decode(&params)
    if err != nil {
        // an error will be thrown if the JSON is invalid or has the wrong types
        // any missing fields will simply have their values in the struct set to their zero value
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(fmt.Sprintf(`{"error": "something went wrong"}`)))
		w.WriteHeader(400)
		return
    }
    // params is a struct with data populated successfully
    // ...
*/
