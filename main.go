package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"internal/database"
	"net/http"
	"strings"
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

//profane word list
var p = []string{"kerfuffle", "sharbert", "fornax"}
var sc = []byte{'!','?',',','.',';',':'}
var id = 1

func replaceProfane(s string) string {
	ns := ""
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(p); j++ {
			if i+len(p[j]) < len(s)  {
				if strings.ToLower(s[i:i+len(p[j])]) == p[j] {
					punctuation := false
					for k := 0; k < len(sc); k++ {
						if s[i+len(p[j])] == sc[k] {
							punctuation = true
							break
						}
					}
					if !punctuation {
						ns = fmt.Sprintf("%s****", ns)
						i += len(p[j])
						break
					}
				}	
			}
			if i+len(p[j]) == len(s) {
				if strings.ToLower(s[i:]) == p[j]{
					ns = fmt.Sprintf("%s****", ns)
					i += len(p[j])				
					break
				}	
			}
		}
		if i < len(s) {
			ns = fmt.Sprintf("%s%s", ns, string(s[i]))
		}
	}
	return ns
}

func (cfg *apiConfig) addChirpHandler(w http.ResponseWriter, r *http.Request) {
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
		w.WriteHeader(201)
		w.Write(msg)
		return
	}
	valid := validateChirpLength(params.Body)
	if !valid {
		err := errors.New("chirp is too long")
		fmt.Printf("Invalid chirp body: %s", err)
		w.WriteHeader(500)
		return
	}
	type successReturnVal struct {
		Id int `json:"id"`
		Body string `json:"body"`
	}

	returnVal := successReturnVal{
		Id: id,
		Body: replaceProfane(params.Body),
	}
	id += 1

	msg, err := json.Marshal(returnVal)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write(msg)
}

func validateChirpLength(c string) (bool) {
	return len(c) <= 140 

	// errorBody := errorReturnVal{
	// 	Error: "Chirp is too long",
	// }
	// msg, err := json.Marshal(errorBody)
	// if err != nil {
	// 	fmt.Printf("Error marshalling JSON: %s", err)
	// 	return
	// }
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(400)
	// w.Write(msg)
}

func main() {
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}
	database.NewDB(".")
	

	h := handler{body:"OK"}
	apiCfg := apiConfig{fileServerHits: 0}
	fs := http.FileServer(http.Dir("."))
	prefixHandler := http.StripPrefix("/app", fs)


	mux.Handle("GET /admin/metrics", http.StripPrefix("/admin/", &apiCfg))
	mux.Handle("GET /api/healthz", h)
	mux.Handle("/api/reset", apiCfg.resetMetrics(h))
	mux.Handle("/app/*", apiCfg.middlewareMetrics(prefixHandler))
	// mux.HandleFunc("POST /api/validate_chirp", apiCfg.validateHandler)
	mux.HandleFunc("POST /api/chirps", apiCfg.addChirpHandler)
	// mux.HandleFunc("GET /api/chirps")

	http.ListenAndServe(srv.Addr, srv.Handler)
}
