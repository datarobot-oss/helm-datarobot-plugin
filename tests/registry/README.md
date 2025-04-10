# Private Registry config


## htpasswd
```
docker run --rm --entrypoint htpasswd httpd:2 -Bbn admin pass
```


## TLS
```
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj "/CN=MyCA"
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/CN=registry" -addext "subjectAltName=DNS:localhost"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 3650 -sha256  -extfile <(printf "subjectAltName=DNS:localhost")

openssl verify -CAfile ca.crt server.crt
openssl x509 -in server.crt -text -noout
```

## Docker

```
docker compose up -d
docker login -u admin -p pass localhost:5000 --tls-verify=false
```
