package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"internal/database"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)	

type handler struct {
	body string
}

type apiConfig struct {
	fileServerHits int
	db *database.DB
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

	decoder := json.NewDecoder(r.Body)
	params := database.Chirp{}

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
		w.WriteHeader(500)
		w.Write(msg)
		return
	}
	valid := validateChirpLength(params.Body)
	if !valid {
		err := errors.New("chirp is too long")
		fmt.Printf("Invalid chirp body: %s", err)
		w.WriteHeader(400)
		return
	}

	chirp, err := cfg.db.CreateChirp(replaceProfane(params.Body))
	if err != nil {
		fmt.Printf("Error creating chirp: %s", err)
		w.WriteHeader(500)
		return
	}

	msg, err := json.Marshal(chirp)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(201)
	w.Write(msg)
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps()
	if err != nil {
		fmt.Printf("Error getting chirps: %s", err)
		w.WriteHeader(500)
		return
	}

	msg, err := json.Marshal(chirps)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	w.Write(msg)
}

func (cfg *apiConfig) getChirpByIDHandler(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("chirpID")

	chirp, err := cfg.db.GetChirpByID(idString)
	if err != nil {
		if err == database.ErrChirpID {
			w.WriteHeader(404)
			return
		}
		fmt.Printf("Error retrieving chirp: %s", err)
		w.WriteHeader(500)
		return
	}
	
	msg, err := json.Marshal(chirp)
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
}

func (cfg *apiConfig) addUserHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := database.User{}

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
		w.WriteHeader(500)
		w.Write(msg)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(params.Password), 10)
	if err != nil {
		fmt.Println("Error generating password hash")
		w.WriteHeader(500)
	}

	user, err := cfg.db.CreateUser(params.Email, string(hashed))
	
	if err != nil {
		fmt.Printf("Error creating user: %s", err)
		w.WriteHeader(500)
		return
	}

	msg, err := json.Marshal(user)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(201)
	w.Write(msg)
}

func (cfg *apiConfig) verifyUserHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := database.User{}

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
		w.WriteHeader(500)
		w.Write(msg)
		return
	}

	user, err := cfg.db.GetUserByEmail(params.Email)
	if err != nil {
		fmt.Printf("User %s not found", params.Email)
		w.WriteHeader(400)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password))
	if err != nil {
		w.WriteHeader(401)
		return
	}

	userNoPass := database.UserNoPassword{
		Email: user.Email,
		ID: user.ID,
	}

	msg, err := json.Marshal(userNoPass)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write(msg)
}

func main() {
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	db, err := database.NewDB(".")
	if err != nil {
		fmt.Println("db error")
	}
	
	h := handler{body:"OK"}
	apiCfg := apiConfig{fileServerHits: 0, db: db}
	fs := http.FileServer(http.Dir("."))
	prefixHandler := http.StripPrefix("/app", fs)


	mux.Handle("GET /admin/metrics", http.StripPrefix("/admin/", &apiCfg))
	mux.Handle("GET /api/healthz", h)
	mux.Handle("/api/reset", apiCfg.resetMetrics(h))
	mux.Handle("/app/*", apiCfg.middlewareMetrics(prefixHandler))
	mux.HandleFunc("POST /api/chirps", apiCfg.addChirpHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.getChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpByIDHandler)
	mux.HandleFunc("POST /api/users", apiCfg.addUserHandler)
	mux.HandleFunc("POST /api/login", apiCfg.verifyUserHandler)

	http.ListenAndServe(srv.Addr, srv.Handler)
}
