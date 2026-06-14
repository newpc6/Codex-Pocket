package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"codexflow/internal/config"
)

type JWTService struct {
	cfg config.Config
}

func NewJWTService(cfg config.Config) *JWTService {
	return &JWTService{cfg: cfg}
}

type Claims struct {
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}

func (j *JWTService) GenerateToken(username string) (string, error) {
	now := time.Now()
	claims := Claims{
		Username: username,
		Exp:      now.Add(24 * time.Hour).Unix(),
		Iat:      now.Unix(),
	}

	header := base64urlEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	payloadEncoded := base64urlEncode(payload)

	signingInput := header + "." + payloadEncoded
	signature := j.sign(signingInput)

	return signingInput + "." + signature, nil
}

func (j *JWTService) ValidateToken(tokenString string) (*Claims, bool) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, false
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSig := j.sign(signingInput)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, false
	}

	payload, err := base64urlDecode(parts[1])
	if err != nil {
		return nil, false
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, false
	}

	if time.Now().Unix() > claims.Exp {
		return nil, false
	}

	return &claims, true
}

func (j *JWTService) sign(input string) string {
	mac := hmac.New(sha256.New, []byte(j.cfg.JWTSecret))
	mac.Write([]byte(input))
	return base64urlEncode(mac.Sum(nil))
}

func base64urlEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func base64urlDecode(s string) ([]byte, error) {
	// Add padding
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
