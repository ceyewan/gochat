package tools

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("your-256-bit-secret")

type Claims struct {
	UserID       int    `json:"user_id"`
	UserName     string `json:"user_name"`
	PasswordHash string `json:"password_hash"`
	jwt.RegisteredClaims
}

// GenerateToken 生成带有过期时间的JWT Token
func GenerateToken(userID int, userName, passwordHash string, expiration time.Duration) (string, error) {
	// 设置过期时间为当前时间加上指定的时长
	expirationTime := jwt.NewNumericDate(time.Now().Add(expiration))

	claims := &Claims{
		UserID:       userID,
		UserName:     userName,
		PasswordHash: passwordHash,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: expirationTime,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken 验证Token（包含过期检查）
func ValidateToken(tokenString string) (int, string, string, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil {
		return -1, "", "", err
	}

	if !token.Valid {
		return -1, "", "", fmt.Errorf("invalid token")
	}

	return claims.UserID, claims.UserName, claims.PasswordHash, nil
}
