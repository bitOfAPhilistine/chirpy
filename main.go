package main

import _ "github.com/lib/pq"
import (
	"os"
	"fmt"
	"net/http"
	"database/sql"
	"github.com/joho/godotenv"
	"github.com/bitOfAPhilistine/chirpy/internal/api"
	"github.com/bitOfAPhilistine/chirpy/internal/database"
)


func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("Error loading database: ", err)
		os.Exit(1)
	}

	dbQueries := database.New(db)

	apiCfg := api.NewConfig(dbQueries)

	mux := http.NewServeMux()

	mux.Handle("/app/", apiCfg.MetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", api.HealthzHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.GetMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.Reset)
	mux.HandleFunc("POST /api/validate_chirp", api.ValidateChirp)

	server := http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	fmt.Println(server.ListenAndServe())
}