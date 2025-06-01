#!/bin/bash
# Generate RSA keypair for Ethos RS256
openssl genpkey -algorithm RSA -out ethos_private.pem -pkeyopt rsa_keygen_bits:2048
openssl rsa -pubout -in ethos_private.pem -out ethos_public.pem

echo "Keys generated:"
echo "  ethos_private.pem (keep secret, used for signing)"
echo "  ethos_public.pem  (safe to distribute, used for verification)"
