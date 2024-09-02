package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"internal/auth"
	"internal/database"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)	

type handler struct {
	body string
}

type apiConfig struct {
	fileServerHits int
	db *database.DB
	jwtsecret string
}

type errorReturnVal struct {
	Error string `json:"error"`
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
	header := r.Header.Get("Authorization")

	bearerToken, err := auth.ParseBearerToken(header)
	if err != nil {
		fmt.Printf("Error parsing bearer token: %s", err)
		w.WriteHeader(401)
		return
	}

	ID, err := auth.ParseUserIDFromJWT(bearerToken)
	if err != nil {
		fmt.Printf("Error validating jwt token: %s", err)
		w.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := database.Chirp{}

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

	chirp, err := cfg.db.CreateChirp(replaceProfane(params.Body), ID)
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
	authorIDString := r.URL.Query().Get("author_id")
	sortParam := r.URL.Query().Get("sort")

	authorID, _ := strconv.Atoi(authorIDString)

	chirps, err := cfg.db.GetChirps(authorID, sortParam)
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

	type loginParams struct {
		database.User
		Expires int `json:"expires_in_seconds"`
	}
	params := loginParams{}
	
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

	refreshToken, err := cfg.db.GenerateRefreshToken(user.ID)
	if err != nil {
		fmt.Printf("Error generating refresh token: %s", err)
		w.WriteHeader(500)
		return
	}

	jwt, err := cfg.db.GenerateAccessToken(refreshToken)
	if err != nil {
		fmt.Printf("Error generating access token: %s", err)
		w.WriteHeader(500)
		return
	}

	userNoPass := database.User{
		Email: user.Email,
		ID: user.ID,
		Token: jwt,
		RefreshToken: refreshToken,
		IsChirpyRed: user.IsChirpyRed,
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

func (cfg *apiConfig) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")

	bearerToken, err := auth.ParseBearerToken(header)
	if err != nil {
		fmt.Printf("Error parsing bearer token: %s", err)
		w.WriteHeader(401)
		return
	}

	ID, err := auth.ParseUserIDFromJWT(bearerToken)
	if err != nil {
		fmt.Printf("Error validating jwt token: %s", err)
		w.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := database.User{}

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

	params.Password = string(hashed)

	user, err := cfg.db.UpdateUser(ID, params)
	if err != nil {
		fmt.Printf("Error updating user: %s", err)
		w.WriteHeader(500)
	}

	userNoPass := database.User{
		Email: user.Email,
		ID: user.ID,
		IsChirpyRed: user.IsChirpyRed,
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

func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")

	bearerToken, err := auth.ParseBearerToken(header)
	if err != nil {
		fmt.Printf("Error parsing bearer token: %s", err)
		w.WriteHeader(401)
		return
	}

	newJWT, err := cfg.db.GenerateAccessToken(bearerToken)
	if err != nil {
		fmt.Printf("Error generating access token: %s", err)
		w.WriteHeader(401)
		return
	}

	type AccessTokenResponse struct {
		Token string `json:"token"`
	}

	msg, err := json.Marshal(AccessTokenResponse{Token:newJWT})
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	w.Write(msg)
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")

	bearerToken, err := auth.ParseBearerToken(header)
	if err != nil {
		fmt.Printf("Error parsing bearer token: %s", err)
		w.WriteHeader(401)
		return
	}

	err = cfg.db.RevokeRefreshToken(bearerToken)
	if err != nil {
		fmt.Printf("Error revoking access token: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) deleteChirpHandler(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")

	bearerToken, err := auth.ParseBearerToken(header)
	if err != nil {
		fmt.Printf("Error parsing bearer token: %s", err)
		w.WriteHeader(401)
		return
	}

	userID, err := auth.ParseUserIDFromJWT(bearerToken)
	if err != nil {
		fmt.Printf("Error validating jwt token: %s", err)
		w.WriteHeader(401)
		return
	}

	chirpIDString := r.PathValue("chirpID")
	chirpID, err := strconv.Atoi(chirpIDString)
	if err != nil {
		fmt.Println("Error converting id")
		w.WriteHeader(401)
		return
	}

	err = cfg.db.DeleteChirpFromDB(userID, chirpID)
	if err != nil {
		fmt.Printf("Error deleting chirp: %s", err)
		w.WriteHeader(403)
		return
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) upgradeUserHandler(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	apikey := strings.Replace(header, "ApiKey ", "", 1)
	if apikey != os.Getenv("POLKA_API_KEY") {
		w.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(r.Body)

	type UpgradeUserParams struct {
		Event string `json:"event"`
		Data struct {
			UserID int `json:"user_id"`
		} `json:"data"`
	}

	params := UpgradeUserParams{}
	
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

	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	err := cfg.db.UpdateChirpyRedStatus(params.Data.UserID, true)
	if errors.Is(err, database.ErrUserNotFound) {
		w.WriteHeader(404)
		return
	}
	if err != nil {
		fmt.Printf("Error updating user chirpy red status: %s", err)
		w.WriteHeader(500)
		return
	}

	type SuccessReturnValue struct {
		Body string `json:"body"`
	}

	msg, err := json.Marshal(SuccessReturnValue{Body:""})
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(204)
	w.Write(msg)

}

func main() {
	err := godotenv.Load()
	if err != nil {
    fmt.Println("Error loading .env file")
  }
	
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
	apiCfg := apiConfig{
		fileServerHits: 0, 
		db: db,
		jwtsecret: os.Getenv("JWT_SECRET_KEY"),
	}
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
	mux.HandleFunc("PUT /api/users", apiCfg.updateUserHandler)
	mux.HandleFunc("POST /api/refresh", apiCfg.refreshHandler)
	mux.HandleFunc("POST /api/revoke", apiCfg.revokeHandler)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.deleteChirpHandler)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.upgradeUserHandler)

	http.ListenAndServe(srv.Addr, srv.Handler)
}
