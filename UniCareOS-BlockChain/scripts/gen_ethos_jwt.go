package main

import (
	"crypto/x509"
	"crypto/rsa"
	"encoding/pem"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <ethos_private.pem>", os.Args[0])
	}
	privPath := os.Args[1]
	privPem, err := ioutil.ReadFile(privPath)
	if err != nil {
		log.Fatal(err)
	}
	block, _ := pem.Decode(privPem)
	if block == nil {
		log.Fatalf("Failed to decode PEM block from %s", privPath)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}
	privKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		log.Fatal("Not an RSA private key")
	}

	claims := jwt.MapClaims{
		"sub":   "1234567890",
		"roles": []string{"admin"},
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privKey)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("JWT:", signed)
}
