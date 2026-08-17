package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)


type ApiConfig struct {
	fileServerHits atomic.Int32
}

func newApiConfig() *ApiConfig {
	newApiConfig := ApiConfig{
		fileServerHits: atomic.Int32{},
	}
	return &newApiConfig
}

func (api *ApiConfig) Reset(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Reset called")
	api.fileServerHits = atomic.Int32{}
	rw.WriteHeader(200)
}

func (api *ApiConfig) MetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Metrics Increment called")
		api.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (api *ApiConfig) GetMetrics(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Get Metrics called")
	req.Header.Add("content-type", "text/plain; charset=utf-8")
	rw.WriteHeader(200)
	rw.Write([]byte(fmt.Sprintf("Hits: %d", api.fileServerHits.Load())))
}


func healthzHandler(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Health Check called")
	req.Header.Add("content-type", "text/plain; charset=utf-8")
	rw.WriteHeader(200)
	rw.Write([]byte("OK"))
}


func main() {
	apiCfg := newApiConfig()

	mux := http.NewServeMux()

	mux.Handle("/app/", apiCfg.MetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /metrics", apiCfg.GetMetrics)
	mux.HandleFunc("POST /reset", apiCfg.Reset)

	server := http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	fmt.Println(server.ListenAndServe())
}