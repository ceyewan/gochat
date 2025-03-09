package tools

import (
	"fmt"
	"gochat/config"
	"sync"
	"time"

	"gochat/clog"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtKey         = []byte(config.Conf.JWTKey.SignKey)
	blacklist      = make(map[string]struct{})
	blacklistMutex sync.Mutex
)

type Claims struct {
	UserID       int    `json:"user_id"`
	UserName     string `json:"user_name"`
	PasswordHash string `json:"password_hash"`
	jwt.RegisteredClaims
}

// GenerateToken 生成带有过期时间的JWT Token
func GenerateToken(userID int, userName, passwordHash string, expiration time.Duration) (string, error) {
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
		clog.Error("Failed to sign token: %v", err)
		return "", err
	}

	clog.Info("Generated token for userID: %d", userID)
	return tokenString, nil
}

// RevokeToken 将 token 加入黑名单
func RevokeToken(tokenString string) {
	blacklistMutex.Lock()
	defer blacklistMutex.Unlock()
	blacklist[tokenString] = struct{}{}
	clog.Info("Token revoked: %s", tokenString)
}

// ValidateToken 验证Token（包含过期检查）, 返回用户ID, 用户名, 密码哈希
func ValidateToken(tokenString string) (int, string, string, error) {
	blacklistMutex.Lock()
	if _, found := blacklist[tokenString]; found {
		blacklistMutex.Unlock()
		clog.Warning("Token is revoked: %s", tokenString)
		return -1, "", "", fmt.Errorf("token is revoked")
	}
	blacklistMutex.Unlock()

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			clog.Error("Unexpected signing method: %v", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil {
		clog.Error("Failed to parse token: %v", err)
		return -1, "", "", err
	}

	if !token.Valid {
		clog.Warning("Invalid token: %s", tokenString)
		return -1, "", "", fmt.Errorf("invalid token")
	}

	clog.Info("Validated token for userID: %d", claims.UserID)
	return claims.UserID, claims.UserName, claims.PasswordHash, nil
}
