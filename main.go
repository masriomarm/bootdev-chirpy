package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
	cfg.fileserverHits.Store(0)
	res.WriteHeader(200) // 200 OK for now.
	res.Write([]byte("OK"))
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
	servMux := http.NewServeMux()
	cfg := apiConfig{}
	servMux.Handle("/app/", cfg.mwMetricInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	servMux.HandleFunc("GET /api/healthz", httpHandler_readiness)
	servMux.HandleFunc("POST /api/validate_chirp", httpHandler_validate_chirp)

	servMux.HandleFunc("GET /admin/metrics", cfg.httpHandler_metricsGet)
	servMux.HandleFunc("POST /admin/reset", cfg.httpHandler_metricsRst)

	server := http.Server{Addr: ":8080", Handler: servMux}
	log.Printf("Starting server: %v", &server)
	log.Fatal(server.ListenAndServe())
}
