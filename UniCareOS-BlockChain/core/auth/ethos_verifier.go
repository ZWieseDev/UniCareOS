package auth

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
)

type EthosClaims struct {
	Sub      string   `json:"sub"`
	ChainID  string   `json:"chainID"`
	Iat      int64    `json:"iat"`
	Exp      int64    `json:"exp"`
	Iss      string   `json:"iss"`
	Roles    []string `json:"roles"`
	Reason   string   `json:"reason"`
	jwt.RegisteredClaims
}

type EthosVerifier struct {
	KeyProvider KeyProvider
}

func (v *EthosVerifier) VerifyEthosToken(tokenString string) (*EthosClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &EthosClaims{}, func(token *jwt.Token) (interface{}, error) {
		kid, _ := token.Header["kid"].(string)
		return v.KeyProvider.GetPublicKey(kid)
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*EthosClaims); ok && token.Valid {
		// Additional claim checks (expiry, chainID, roles, etc.) can go here
		return claims, nil
	}
	return nil, errors.New("invalid ethos token or claims")
}
