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
	serveMux.HandleFunc("POST /api/chirps", createChirpHandler)
	serveMux.HandleFunc("POST /api/users", createUserHandler) 
	//serveMux.HandleFunc("GET /api/chirps", getAllChirpsHandler)
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

func createChirpHandler(w http.ResponseWriter, r *http.Request) {
    // Define request structure
    type ChirpRequest struct {
        Body   string `json:"body"`
        UserID string `json:"user_id"`
    }

    // Parse request body
    var req ChirpRequest
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
        return
    }

    // Validate chirp length
    if len(req.Body) > 140 {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "Chirp is too long"})
        return
    }

    // Clean profanity if present
    cleanedBody := req.Body
    if containsProfanity(req.Body) {
        cleanedBody = cleanBody(req.Body)
    }

    // Parse UserID from string to UUID
    userID, err := uuid.Parse(req.UserID)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "Invalid user ID"})
        return
    }

    // Create chirp in database
	params := database.CreateChirpParams{
        Body:     	cleanedBody,
        UserID: 	userID,
    }
    
    chirp, err := apiCfg.DB.CreateChirp(r.Context(), params)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create chirp"})
        return
    }

    // Format response
    type ChirpResponse struct {
        ID        string    `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
        Body      string    `json:"body"`
        UserID    string    `json:"user_id"`
    }

    resp := ChirpResponse{
        ID:        chirp.ID.String(),
        CreatedAt: chirp.CreatedAt,
        UpdatedAt: chirp.UpdatedAt,
        Body:      chirp.Body,
        UserID:    chirp.UserID.String(),
    }

    // Return successful response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(resp)
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
	w.WriteHeader(http.StatusCreated)
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

type jsonChirp struct {
	ID        	uuid.UUID 		`json:"id"`
	CreatedAt 	time.Time 		`json:"created_at"`
	UpdatedAt 	time.Time 		`json:"updated_at"`
	Body   		string			`json:"body"`
	UserID 		uuid.UUID		`json:"userID"`
}
/*
func getAllChirpsHandler(w http.ResponseWriter, r *http.Request) {
	type ChirpResponse struct {
        ID        string    `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
        Body      string    `json:"body"`
        UserID    string    `json:"user_id"`
    }

    resp := ChirpResponse{
        ID:        chirp.ID.String(),
        CreatedAt: chirp.CreatedAt,
        UpdatedAt: chirp.UpdatedAt,
        Body:      chirp.Body,
        UserID:    chirp.UserID.String(),
    }
} 
*/
/*
func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		log.Println(err)
	}
	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
} 
*/