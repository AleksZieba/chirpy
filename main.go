package main

import (
	"log"
	"net/http"
	"fmt" 
	"sync/atomic" 
	"strconv" 
	"io" 
	"encoding/json" 
	"time" 
	"strings" 
	_ "github.com/lib/pq" 
	"database/sql"
	"github.com/joho/godotenv" 
	"os"
	"github.com/AleksZieba/chirpy/internal/database"
	"github.com/google/uuid"
	//"context"
)

func main() { 
	godotenv.Load()
	dbURL := os.Getenv("DB_URL") 
	db, err := sql.Open("postgres", dbURL) 
	if err != nil {
		log.Fatal(err)
	} 
	
	dbQueries := database.New(db)
	modApiConfig(&apiCfg, *dbQueries)

	s := initServer()
	log.Fatal(s.ListenAndServe())
} 

type apiConfig struct {
	fileserverHits 	atomic.Int32 
	DB 				database.Queries
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
	serveMux.HandleFunc("POST /api/users", createUserHandler)
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

func modApiConfig(aC *apiConfig, dbQ database.Queries) {
	aC.DB = dbQ
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
	//w.WriteHeader(http.StatusOK) 
	apiCfg.fileserverHits.Store(0)
	err := apiCfg.DB.DeleteAllUsers(r.Context())
	if err != nil {
		log.Fatalf("error reseting users table: %v\n", err)
	} 
	w.WriteHeader(http.StatusOK)
}  

type profaneWords struct {
	wordlist	[]string
} 

var badWords = []string{"kerfuffle", "sharbert", "fornax",}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
        Body string `json:"body"` 
	}
	
	type returnVals struct {
        //CreatedAt time.Time `json:"created_at"`
        //ID int `json:"id"` 
		//Valid bool `json:"valid"`
		Body string `json:"cleaned_body"`
    } 

	respBody := returnVals{}
		//CreatedAt: time.Now(),
		//ID: 123, 
		//Valid: true,
	//}

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
	} else if containsProfanity(params.Body) {
		respBody.Body = cleanBody(params.Body)
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
	} else { 
		respBody.Body = params.Body
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

func containsProfanity(reqBody string) bool {
	//bodySlice := strings.Split(strings.ToLower(reqBody), " ") 
	if strings.Contains(reqBody, "kerfuffle") == true {
		return true 
	}  
	if strings.Contains(reqBody, "sharbert") == true {
		return true 
	} 
	if strings.Contains(reqBody, "fornax") == true {
		return true 
	} 
	return false
}

func cleanBody(reqBody string) string {
	bodySlice := strings.Split(reqBody, " ")
	bodySliceLower := strings.Split(strings.ToLower(reqBody), " ")
	for i, word := range(bodySliceLower) {
		if word == "" {
			continue
		} 
		for _, badword := range(badWords) {
			if badword == word {
			bodySlice[i] = "****"
			}
		} 
	}
	return strings.Join(bodySlice, " ")	
}

type jsonUser struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	newUser := jsonUser{}
	
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
	err = json.Unmarshal(bytesBody, &newUser)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(fmt.Sprintf(`{"error": "something went wrong"}`))) 
			if err != nil {
				log.Fatal("failed to write response")
			}
			//w.WriteHeader(400)
			return
		} 
	user, err := apiCfg.DB.CreateUser(r.Context(), newUser.Email) 
		if err != nil {
			log.Fatalf("failed to create user: %v", err) 
			return
		} 
	jsonU := dbUserToMarshallingUser(user) 
	dat, err := json.Marshal(jsonU)
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

func dbUserToMarshallingUser(dbUser database.User) jsonUser {
	return jsonUser {
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
}