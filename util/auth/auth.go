package auth

import (
	"fmt"
	"sniper/util/conf"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func secret() string {
	return conf.Get("JWT_SECRET")
}

func Generate(userId string, now time.Time, expire time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"usr": userId,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(expire).Unix(),
	})
	tokenString, err := token.SignedString([]byte(secret()))

	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func Authenticate(tokenString string) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return false, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(secret()), nil
	})

	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return true, nil
	}
	return false, err
}
