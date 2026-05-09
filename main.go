package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/masriomarm/bootdev-chirpy/internal/database"
)

func err_response(res http.ResponseWriter, errMsg string, statucCode int) {

	type errBody struct {
		Error string `json:"error"`
	}

	log.Printf(errMsg)
	dat, err := json.Marshal(errBody{Error: errMsg})
	if err != nil {
		log.Printf("Error while sending error ... LOL : %v", err)
	}
	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(statucCode)
	res.Write(dat)
}

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

func (cfg *apiConfig) mwMetricInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) httpHandler_metricsGet(res http.ResponseWriter, req *http.Request) {
	templateResponce := `
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`
	counter := fmt.Sprintf(templateResponce, cfg.fileserverHits.Load())
	res.Write([]byte(counter))
}

func (cfg *apiConfig) httpHandler_metricsRst(res http.ResponseWriter, req *http.Request) {
	statusCode := 200
	msgResponse := "OK"
	if cfg.platform == "dev" {
		// reset server hits
		cfg.fileserverHits.Store(0)

		// delete all users
		err := cfg.db.DeleteUsers(req.Context())
		if err != nil {
			log.Printf("Error deleting users: %v", err)
			statusCode = 500
			msgResponse = "Error"
		}

	} else {
		statusCode = 403
		msgResponse = "Forbidden"
	}
	res.WriteHeader(statusCode)
	res.Write([]byte(msgResponse))
}

func (cfg *apiConfig) httpHandler_chirpCreate(res http.ResponseWriter, req *http.Request) {
	type reqBody struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	type resBody struct {
		Id          uuid.UUID `json:"id"`
		CreatedTime time.Time `json:"created_at"`
		UpdatedTime time.Time `json:"updated_at"`
		Body        string    `json:"body"`
		UserId      uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(req.Body)
	input := reqBody{}

	err := decoder.Decode(&input)
	if err != nil {
		errMsg := fmt.Sprintf("Error decoding parameters: %v", err)
		err_response(res, errMsg, http.StatusBadRequest)
		return
	}

	if app_chirp_valid(input.Body) == false {
		err_response(res, "Chirp is too long", http.StatusBadRequest)
		return
	}

	chirp, err := cfg.db.CreateChirp(req.Context(), database.CreateChirpParams{Body: app_chirp_profane_mask(input.Body), UserID: input.UserID})
	if err != nil {
		errMsg := fmt.Sprintf("Error creating query: %v", err)
		err_response(res, errMsg, http.StatusInternalServerError)
		return
	}

	ret := resBody{Id: chirp.ID, CreatedTime: chirp.CreatedAt, UpdatedTime: chirp.UpdatedAt, Body: chirp.Body, UserId: chirp.UserID}
	dat, err := json.Marshal(ret)
	if err != nil {
		errMsg := fmt.Sprintf("Error marshalling JSON: %s", err)
		err_response(res, errMsg, http.StatusInternalServerError)
		return
	}

	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	res.Write(dat)
}

func (cfg *apiConfig) httpHandler_chirpGetByID(res http.ResponseWriter, req *http.Request) {

	type resBody struct {
		Id          uuid.UUID `json:"id"`
		CreatedTime time.Time `json:"created_at"`
		UpdatedTime time.Time `json:"updated_at"`
		Body        string    `json:"body"`
		UserId      uuid.UUID `json:"user_id"`
	}

	// revert `input` to uuid to be passed to db
	input := req.PathValue("chirpID")
	id, err := uuid.Parse(input)
	if err != nil {
		err_response(res, "Invalid chirp", http.StatusBadRequest)
		return
	}

	chirp, err := cfg.db.GetChirpByID(req.Context(), id)
	if err != nil {
		errMsg := fmt.Sprintf("Error creating query: %v", err)
		err_response(res, errMsg, http.StatusNotFound)
		return
	}

	ret := resBody{Id: chirp.ID, CreatedTime: chirp.CreatedAt, UpdatedTime: chirp.UpdatedAt, Body: chirp.Body, UserId: chirp.UserID}
	dat, err := json.Marshal(ret)
	if err != nil {
		errMsg := fmt.Sprintf("Error marshalling JSON: %s", err)
		err_response(res, errMsg, http.StatusInternalServerError)
		return
	}

	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(dat)
}

func (cfg *apiConfig) httpHandler_chirpGet(res http.ResponseWriter, req *http.Request) {
	type reqBody struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	type resBody struct {
		Id          uuid.UUID `json:"id"`
		CreatedTime time.Time `json:"created_at"`
		UpdatedTime time.Time `json:"updated_at"`
		Body        string    `json:"body"`
		UserId      uuid.UUID `json:"user_id"`
	}

	chirps, err := cfg.db.GetChirp(req.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error creating query: %v", err)
		err_response(res, errMsg, http.StatusInternalServerError)
		return
	}

	ret := make([]resBody, 0, len(chirps))
	for _, chirp := range chirps {
		dbChirp := resBody{Id: chirp.ID, CreatedTime: chirp.CreatedAt, UpdatedTime: chirp.UpdatedAt, Body: chirp.Body, UserId: chirp.UserID}
		ret = append(ret, dbChirp)
	}

	dat, err := json.Marshal(ret)
	if err != nil {
		errMsg := fmt.Sprintf("Error marshalling JSON: %s", err)
		err_response(res, errMsg, http.StatusInternalServerError)
		return
	}

	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(dat)
}

func (cfg *apiConfig) httpHandler_userCreate(res http.ResponseWriter, req *http.Request) {
	type reqBody struct {
		Email string `json:"email"`
	}

	type resBody struct {
		Id          uuid.UUID `json:"id"`
		CreatedTime time.Time `json:"created_at"`
		UpdatedTime time.Time `json:"updated_at"`
		Email       string    `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	input := reqBody{}

	err := decoder.Decode(&input)
	statusCode := 500
	if err != nil {
		log.Printf("Error decoding parameters: %v", err)
		res.WriteHeader(statusCode)
		return
	}

	user, err := cfg.db.CreateUser(req.Context(), input.Email)
	if err != nil {
		log.Printf("Error creating query: %v", err)
		res.WriteHeader(statusCode)
		return
	}

	ret := resBody{Id: user.ID, CreatedTime: user.CreatedAt, UpdatedTime: user.UpdatedAt, Email: user.Email}
	dat, err := json.Marshal(ret)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		res.WriteHeader(statusCode)
		return
	}

	statusCode = 201 // user created
	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(statusCode)
	res.Write(dat)
}

func httpHandler_readiness(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200) // 200 OK for now.
	res.Write([]byte("OK"))
}

func app_chirp_profane_mask(chirp string) string {
	words := strings.Split(chirp, " ")
	profaneWords := map[string]struct{}{"kerfuffle": {}, "sharbert": {}, "fornax": {}}
	for indx, word := range words {
		key := strings.ToLower(word)
		if _, ok := profaneWords[key]; ok {
			words[indx] = "****"
		}
	}
	return strings.Join(words, " ")
}

func app_chirp_valid(chirp string) bool {
	ret := true
	if len(chirp) > 140 {
		ret = false
	}
	return ret
}

func httpHandler_validate_chirp(res http.ResponseWriter, req *http.Request) {
	type msgBody struct {
		Body string
	}

	type responseBody struct {
		Valid       *bool   `json:"valid,omitempty"`
		Error       *string `json:"error,omitempty"`
		Body        *string `json:"body,omitempty"`
		CleanedBody *string `json:"cleaned_body,omitempty"`
	}

	decoder := json.NewDecoder(req.Body)
	chirp := msgBody{}

	err := decoder.Decode(&chirp)
	if err != nil {
		log.Printf("Error decoding parameters: %v", err)
		res.WriteHeader(500)
		return
	}

	var ret responseBody
	statusCode := 200
	if app_chirp_valid(chirp.Body) == true {
		body := app_chirp_profane_mask(chirp.Body)
		ret.CleanedBody = &body
	} else {
		errorMsg := "Chirp is too long"
		ret.Error = &errorMsg
		statusCode = 400
	}

	dat, err := json.Marshal(ret)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		res.WriteHeader(500)
		return
	}

	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(statusCode)
	res.Write(dat)
}

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error init database: %v", err)
	}

	dbQueries := database.New(db)
	platform := os.Getenv("PLATFORM")
	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		platform:       platform,
	}

	servMux := http.NewServeMux()
	servMux.Handle("/app/", cfg.mwMetricInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	servMux.HandleFunc("GET /api/healthz", httpHandler_readiness)
	servMux.HandleFunc("POST /api/validate_chirp", httpHandler_validate_chirp)

	servMux.HandleFunc("GET /admin/metrics", cfg.httpHandler_metricsGet)
	servMux.HandleFunc("POST /admin/reset", cfg.httpHandler_metricsRst)

	servMux.HandleFunc("POST /api/users", cfg.httpHandler_userCreate)
	servMux.HandleFunc("POST /api/chirps", cfg.httpHandler_chirpCreate)
	servMux.HandleFunc("GET /api/chirps", cfg.httpHandler_chirpGet)
	servMux.HandleFunc("GET /api/chirps/{chirpID}", cfg.httpHandler_chirpGetByID)

	server := http.Server{Addr: ":8080", Handler: servMux}
	log.Printf("Starting server: %v", &server)
	log.Fatal(server.ListenAndServe())
}
