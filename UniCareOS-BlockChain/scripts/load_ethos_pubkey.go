package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"log"
)

func LoadRSAPublicKeyFromFile(path string) *rsa.PublicKey {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read public key: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PUBLIC KEY" {
		log.Fatalf("Failed to decode PEM block containing public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse public key: %v", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		log.Fatalf("Not an RSA public key")
	}
	return rsaPub
}
