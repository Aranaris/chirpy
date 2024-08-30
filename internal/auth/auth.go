package auth

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func ParseBearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("No jwt in header")
	}

	header = strings.Replace(header, "Bearer ", "", 1)
	return header, nil
}

func ParseUserIDFromJWT(jsonWebToken string) (int, error) {
	token, err := jwt.ParseWithClaims(jsonWebToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
    return []byte(os.Getenv("JWT_SECRET_KEY")), nil
	})
	if err != nil {
		return 0, err
	}

	userID, err := token.Claims.GetSubject()
	if err != nil {
		return 0, err
	}

	ID, err := strconv.Atoi(userID)
	if err != nil {
		return 0, err
	}

	return ID, nil
}
