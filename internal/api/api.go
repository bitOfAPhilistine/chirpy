package api

import (
	"fmt"
	"time"
	"strings"
	"net/http"
	"sync/atomic"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/bitOfAPhilistine/chirpy/internal/database"
)


type ApiConfig struct {
	fileServerHits atomic.Int32
	dbQueries *database.Queries
	platform string
}

func NewConfig(queries *database.Queries, platform string) *ApiConfig {
	newConfig := ApiConfig{
		fileServerHits: atomic.Int32{},
		dbQueries: queries,
		platform: platform,
	}
	return &newConfig
}

func (api *ApiConfig) Reset(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Reset called")
	if api.platform != "dev" {
		rw.WriteHeader(403)
		return
	}

	api.fileServerHits = atomic.Int32{}
	if err := api.dbQueries.ClearUsers(req.Context()); err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		return
	}
	if err := api.dbQueries.ClearChirps(req.Context()); err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		return
	}
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


func decodeRequestToJson(rw http.ResponseWriter, req *http.Request, res any) error {
	if err := json.NewDecoder(req.Body).Decode(res); err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		return err
	}
	return nil
}

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

type Chirp struct {
	ID        uuid.UUID		`json:"id"`
	CreatedAt time.Time		`json:"created_at"`
	UpdatedAt time.Time		`json:"updated_at"`
	Body      string		`json:"body"`
	UserID    uuid.NullUUID `json:"user_id"`
}

func (api *ApiConfig) CreateChirp(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Creating chirp:\n", req.Body)
	rw.Header().Set("Content-Type", "application/json")

	res := struct {
		Body string `json:"body"`
		UserId uuid.NullUUID `json:"user_id"`
	}{}
	if decodeRequestToJson(rw, req, &res) != nil {return}

	fmt.Println(res.Body)
	length := len(res.Body)
	if length > 140 {
		rw.WriteHeader(400)
		rw.Write([]byte("{\"Error\":\"Chirp is too long\"}"))
		return
	}

	spl := strings.Split(string(res.Body), " ")
	for i, word := range spl {
		for _, prof := range profanity {
			if strings.ToLower(word) == prof {
				spl[i] = "****"
				break
			}
		}
	}
	res.Body = strings.Join(spl, " ")

	chirp, err := api.dbQueries.CreateChirp(req.Context(), database.CreateChirpParams{
		Body: res.Body,
		UserID: res.UserId,
	})
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		return
	}

	out, err := json.Marshal(Chirp(chirp))
	if err != nil{
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		return
	}

	rw.WriteHeader(201)
	rw.Write(out)
}

func (api *ApiConfig) GetChirps(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Getting all chirps")
	rw.Header().Set("Content-Type", "application/json")

	chirps, err := api.dbQueries.GetChirps(req.Context())
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		return
	}

	outChirps := make([]Chirp, len(chirps))
	for i, chirp := range chirps {
		outChirps[i] = Chirp(chirp)
	}

	out, err := json.Marshal(outChirps)
	if err != nil{
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		return
	}

	rw.WriteHeader(200)
	rw.Write(out)
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (api *ApiConfig) CreateUser(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Creating user")
	rw.Header().Set("Content-Type", "application/json")

	res := struct {Email string `json:"email"`}{}
	if decodeRequestToJson(rw, req, &res) != nil {return}

	user, err := api.dbQueries.CreateUser(req.Context(), res.Email)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		return
	}
	
	out, err := json.Marshal(User(user))
	if err != nil{
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		return
	}

	rw.WriteHeader(201)
	rw.Write(out)
}