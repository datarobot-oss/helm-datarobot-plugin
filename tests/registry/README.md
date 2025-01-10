# Private Registry config


## htpasswd
```
docker run --rm --entrypoint htpasswd httpd:2 -Bbn admin pass
```


## TLS
```
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -sha256
```

## Docker

```
docker login -u admin -p pass localhost:8443 --tls-verify=false
```
