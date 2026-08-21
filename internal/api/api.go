package api

import (
	"fmt"
	"strings"
	"net/http"
	"sync/atomic"
	"encoding/json"
	"github.com/bitOfAPhilistine/chirpy/internal/database"
)


func HealthzHandler(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Health Check called")
	req.Header.Add("content-type", "text/plain; charset=utf-8")
	rw.WriteHeader(200)
	rw.Write([]byte("OK"))
}

var profanity = [...]string{
	"kerfuffle",
	"sharbert",
	"fornax",
}

func ValidateChirp(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Validating post:\n", req.Body)
	rw.Header().Set("Content-Type", "application/json")

	dec := json.NewDecoder(req.Body)
	res := struct {
		Body string `json:"body"`
	}{}
	if err := dec.Decode(&res); err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		return
	}

	length := len(res.Body)
	fmt.Println(length, " chars long")
	if length < 140 {
		spl := strings.Split(string(res.Body), " ")
		for i, word := range spl {
			for _, prof := range profanity {
				if strings.ToLower(word) == prof {
					spl[i] = "****"
					break
				}
			}
		}
    	rw.WriteHeader(200)
    	rw.Write([]byte(fmt.Sprintf("{\"cleaned_body\":\"%s\"}", strings.Join(spl, " "))))
		return
	}

	rw.WriteHeader(400)
	rw.Write([]byte("{\"Error\":\"Chirp is too long\"}"))
}


type ApiConfig struct {
	fileServerHits atomic.Int32
	DbQueries *database.Queries
}

func NewConfig(queries *database.Queries) *ApiConfig {
	newConfig := ApiConfig{
		fileServerHits: atomic.Int32{},
		DbQueries: queries,
	}
	return &newConfig
}

func (api *ApiConfig) Reset(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Reset called")
	api.fileServerHits = atomic.Int32{}
	rw.WriteHeader(200)
}

func (api *ApiConfig) GetMetrics(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Get Metrics called")
	req.Header.Add("content-type", "text/html; charset=utf-8")
	rw.WriteHeader(200)
	rw.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, api.fileServerHits.Load())))
}

func (api *ApiConfig) MetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Metrics Increment called")
		api.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}