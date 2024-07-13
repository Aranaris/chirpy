package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)	

type handler struct {
	body string
}

type apiConfig struct {
	fileServerHits int
}

func outputHTML(w http.ResponseWriter, filename string, data interface{}) {
	t, err := template.ParseFiles(filename)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err:= t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := map[string]interface{}{"FileServerHits": fmt.Sprintf("%d", cfg.fileServerHits)}
	outputHTML(w, "api/metrics/index.html", data)
}

func (cfg *apiConfig) resetMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			cfg.fileServerHits = 0
	})
}

func (cfg *apiConfig) validateHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}

	type errorReturnVal struct {
		Error string `json:"error"`
	}
	
	if err := decoder.Decode(&params); err != nil {
		errorBody := errorReturnVal{
			Error: "Something went wrong",
		}

		msg, err := json.Marshal(errorBody)
		if err != nil {
			fmt.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}

		fmt.Printf("Error Decoding JSON: %s", err)
		w.Write(msg)
		return
	}

	if len(params.Body) <= 140 {
		type successReturnVal struct {
			Valid bool `json:"valid"`
		}

		returnVal := successReturnVal{
			Valid: true,
		}

		msg, err := json.Marshal(returnVal)
		if err != nil {
			fmt.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		w.Write(msg)
		return
	}
	
	errorBody := errorReturnVal{
		Error: "Chirp is too long",
	}
	msg, err := json.Marshal(errorBody)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	w.Write(msg)
}

func main() {
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	h := handler{body:"OK"}
	apiCfg := apiConfig{fileServerHits: 0}
	fs := http.FileServer(http.Dir("."))
	prefixHandler := http.StripPrefix("/app", fs)

	mux.Handle("GET /admin/metrics/", http.StripPrefix("/admin/", &apiCfg))
	mux.Handle("GET /api/healthz/", h)
	mux.Handle("/api/reset/", apiCfg.resetMetrics(h))
	mux.Handle("/app/*", apiCfg.middlewareMetrics(prefixHandler))
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.validateHandler)

	http.ListenAndServe(srv.Addr, srv.Handler)
}
