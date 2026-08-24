package api

import (
	"fmt"
	"time"
	"strings"
	"net/http"
	"sync/atomic"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/bitOfAPhilistine/chirpy/internal/auth"
	"github.com/bitOfAPhilistine/chirpy/internal/database"
)


type ApiConfig struct {
	fileServerHits atomic.Int32
	dbQueries *database.Queries
	platform string
	secretKey string
	polkaKey string
}

func NewConfig(queries *database.Queries, platform string, secretKey string, polkaKey string) *ApiConfig {
	newConfig := ApiConfig{
		fileServerHits: atomic.Int32{},
		dbQueries: queries,
		platform: platform,
		secretKey: secretKey,
		polkaKey: polkaKey,
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
	rw.Write([]byte(fmt.Sprintf(`<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>`, api.fileServerHits.Load())))
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
	UserID    uuid.UUID `json:"user_id"`
}

func (api *ApiConfig) CreateChirp(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Creating chirp:\n", req.Body)
	rw.Header().Add("Content-Type", "application/json")

	res := struct {
		Body string `json:"body"`
	}{}
	if decodeRequestToJson(rw, req, &res) != nil {return}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		rw.WriteHeader(401)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	userId, err := auth.ValidateJWT(token, api.secretKey)
	if err != nil {
		rw.WriteHeader(401)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

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
		UserID: userId,
	})
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	out, err := json.Marshal(Chirp(chirp))
	if err != nil{
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rw.WriteHeader(201)
	rw.Write(out)
}

func (api *ApiConfig) GetChirps(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	var chirps []database.Chirp
	var err error
	authorIDStr := req.URL.Query().Get("author_id")
	if authorIDStr == "" {
		fmt.Println("Getting all chirps")

		chirps, err = api.dbQueries.GetChirps(req.Context())
		if err != nil {
			rw.WriteHeader(500)
			rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
			fmt.Println(err)
			return
		}
	} else {
		fmt.Println("Getting all chirps from user ", authorIDStr)

		authorID, err := uuid.Parse(authorIDStr)
		if err != nil {
			rw.WriteHeader(500)
			rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
			fmt.Println(err)
			return
		}

		chirps, err = api.dbQueries.GetChirpsByAuthor(req.Context(), authorID)
		if err != nil {
			rw.WriteHeader(404)
			rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
			fmt.Println(err)
			return
		}
	}

	outChirps := make([]Chirp, len(chirps))
	for i, chirp := range chirps {
		outChirps[i] = Chirp(chirp)
	}

	out, err := json.Marshal(outChirps)
	if err != nil{
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rw.WriteHeader(200)
	rw.Write(out)
}

func (api *ApiConfig) GetChirp(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Getting chirp")
	id, err := uuid.Parse(req.PathValue("id"))
	if err != nil {
		rw.WriteHeader(404)
		rw.Write([]byte("Error: Post not found"))
		fmt.Println(err)
		return
	}

	chirp, err := api.dbQueries.GetChirp(req.Context(), id)
	if err != nil {
		rw.WriteHeader(404)
		rw.Write([]byte("Error: Post not found"))
		fmt.Println(err)
		return
	}

	out, err := json.Marshal(Chirp(chirp))
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rw.WriteHeader(200)
	rw.Write(out)
}

func (api *ApiConfig) DeleteChirp(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Deleting chirp")

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		rw.WriteHeader(401)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	userID, err := auth.ValidateJWT(token, api.secretKey)
	if err != nil {
		rw.WriteHeader(401)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	chirpID, err := uuid.Parse(req.PathValue("id"))
	if err != nil {
		rw.WriteHeader(404)
		rw.Write([]byte("Error: Post not found"))
		fmt.Println(err)
		return
	}

	chirp, err := api.dbQueries.GetChirp(req.Context(), chirpID)
	if err != nil {
		rw.WriteHeader(404)
		rw.Write([]byte("Error: Post not found"))
		fmt.Println(err)
		return
	}

	if chirp.UserID != userID {
		rw.WriteHeader(403)
		return
	}

	if err = api.dbQueries.DeleteChirp(req.Context(), chirpID); err != nil {
		rw.WriteHeader(403)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rw.WriteHeader(204)
}

type User struct {
	ID        		uuid.UUID 	`json:"id"`
	CreatedAt 		time.Time 	`json:"created_at"`
	UpdatedAt 		time.Time 	`json:"updated_at"`
	Email     		string    	`json:"email"`
	HashedPassword 	string	  	`json:"-"`
	IsChirpyRed    	bool		`json:"is_chirpy_red"`
}

type Login struct {
	Email 		string 	`json:"email"`
	Password 	string 	`json:"password"`
}

func (api *ApiConfig) CreateUser(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Creating user")
	rw.Header().Add("Content-Type", "application/json")

	res := Login{}
	if decodeRequestToJson(rw, req, &res) != nil {return}
	fmt.Println(res.Email)

	hashedPassword, err := auth.HashPassword(res.Password)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	user, err := api.dbQueries.CreateUser(req.Context(), database.CreateUserParams{
		Email: res.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	out, err := json.Marshal(User(user))
	if err != nil{
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rw.WriteHeader(201)
	rw.Write(out)
}

func (api *ApiConfig) Login(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Logging in user")
	rw.Header().Add("Content-Type", "application/json")

	res := Login{}
	if decodeRequestToJson(rw, req, &res) != nil {return}
	fmt.Println(res.Email)

	user, err := api.dbQueries.GetUserByEmail(req.Context(), res.Email)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	passCheck, err := auth.CheckPasswordHash(res.Password, user.HashedPassword)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	} else if !passCheck {
		rw.WriteHeader(401)
		rw.Write([]byte("Incorrect email or password"))
		return
	}

	token, err := auth.MakeJWT(user.ID, api.secretKey, time.Hour)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	refreshToken, err := api.dbQueries.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
		Token: auth.MakeRefreshToken(),
		UserID: user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	})
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	type UserWithTokens struct {
		User
		Token string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	userWithTokens := UserWithTokens{
		User: User(user),
		Token: token,
		RefreshToken: refreshToken,
	}
	
	out, err := json.Marshal(userWithTokens)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rw.WriteHeader(200)
	rw.Write(out)
}

func (api *ApiConfig) ChangeLogin(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Changing user login")
	rw.Header().Add("Content-Type", "application/json")

	res := Login{}
	if decodeRequestToJson(rw, req, &res) != nil {return}
	fmt.Println(res.Email)

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		rw.WriteHeader(401)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	userID, err := auth.ValidateJWT(token, api.secretKey)
	if err != nil {
		rw.WriteHeader(401)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	hashedPassword, err := auth.HashPassword(res.Password)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	user, err := api.dbQueries.ChangeLogin(req.Context(), database.ChangeLoginParams{
		ID: userID,
		Email: res.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	out, err := json.Marshal(User(user))
	if err != nil{
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rw.WriteHeader(200)
	rw.Write(out)
}

func (api *ApiConfig) Refresh(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Refreshing user access token")
	rw.Header().Add("Content-Type", "application/json")

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		rw.WriteHeader(401)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rt, err := api.dbQueries.GetRefreshToken(req.Context(), token)
	if err != nil {
		rw.WriteHeader(401)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	} else if time.Now().Compare(rt.ExpiresAt) >= 0 {
		rw.WriteHeader(401)
		rw.Write([]byte("Refresh token expired"))
		fmt.Println("Refresh token expired")
		return
	} else if rt.RevokedAt.Valid {
		rw.WriteHeader(401)
		rw.Write([]byte("Refresh token revoked"))
		fmt.Println("Refresh token revoked")
		return
	}

	newToken, err := auth.MakeJWT(rt.UserID, api.secretKey, time.Hour)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rw.WriteHeader(200)
	rw.Write([]byte(fmt.Sprintf("{\"token\": \"%s\"}", newToken)))
}

func (api *ApiConfig) Revoke(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Revoking user access token")

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		rw.WriteHeader(401)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	if err = api.dbQueries.RevokeRefreshToken(req.Context(), token); err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rw.WriteHeader(204)
}

func (api *ApiConfig) UpgradeUser(rw http.ResponseWriter, req *http.Request) {
	apiKey, err := auth.GetAPIKey(req.Header)
	if err != nil || apiKey != api.polkaKey {
		rw.WriteHeader(401)
		fmt.Println("Invalid or no Polka API Key given")
		return
	}

	res := struct {
		Event string `json:"event"`
		Data struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}{}
	if decodeRequestToJson(rw, req, &res) != nil {return}

	if res.Event != "user.upgraded" {
		rw.WriteHeader(204)
		fmt.Println(fmt.Sprintf("Unhandled polka webhook request: %s", res.Event))
		return
	}

	userID, err := uuid.Parse(res.Data.UserID)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	user, err := api.dbQueries.UpgradeUser(req.Context(), userID)
	if err != nil {
		rw.WriteHeader(404)
		rw.Write([]byte(fmt.Sprintf("{\"Error\":\"%s\"}", err)))
		fmt.Println(err)
		return
	}

	rw.WriteHeader(204)
	fmt.Printf("User %s upgraded to Chirpy Red\n", user.Email)
}