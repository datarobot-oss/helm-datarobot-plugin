#!/bin/bash
echo "generating CA"
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj "/CN=MyCA"
echo "generating server cert"
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/CN=registry" -addext "subjectAltName=DNS:localhost"
echo "signing server cert"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 3650 -sha256  -extfile <(printf "subjectAltName=DNS:localhost")
