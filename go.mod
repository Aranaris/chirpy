module github.com/aranaris/chirpy

go 1.22.0

require internal/database v1.0.0

replace internal/database => ./internal/database

require (
	github.com/joho/godotenv v1.5.1 // indirect
	golang.org/x/crypto v0.26.0 // indirect
)
