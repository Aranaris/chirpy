package main

import (
	"fmt"
	"net/http"
)	

type handler struct {
	body string
}

type apiConfig struct {
	fileServerHits int
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(h.body))
}

func (cfg *apiConfig) middlewareMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			cfg.fileServerHits++
		})
}

func (cfg *apiConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	output := fmt.Sprintf("Hits: %d", cfg.fileServerHits)
	w.Write([]byte(output))
}

func (cfg *apiConfig) resetMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			cfg.fileServerHits = 0
	})
}

func main() {
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	h := handler{body:"Welcome to Chirpy"}
	apiCfg := apiConfig{fileServerHits: 0}

	mux.Handle("/", http.FileServer(http.Dir('.')))
	mux.Handle("/metrics/*", &apiCfg)
	mux.Handle("/app/*", apiCfg.middlewareMetrics(h))
	mux.Handle("/reset/*", apiCfg.resetMetrics(h))
	mux.Handle("/healthz", h)

	http.ListenAndServe(srv.Addr, srv.Handler)
}
