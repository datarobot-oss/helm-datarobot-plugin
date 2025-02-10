package cmd

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestGetTransport(t *testing.T) {
	// Create temporary files for testing
	caCertFile, err := ioutil.TempFile("", "ca-cert.pem")
	if err != nil {
		t.Fatalf("failed to create temp CA cert file: %v", err)
	}
	defer os.Remove(caCertFile.Name())

	clientCertFile, err := ioutil.TempFile("", "client-cert.pem")
	if err != nil {
		t.Fatalf("failed to create temp client cert file: %v", err)
	}
	defer os.Remove(clientCertFile.Name())

	clientKeyFile, err := ioutil.TempFile("", "client-key.pem")
	if err != nil {
		t.Fatalf("failed to create temp client key file: %v", err)
	}
	defer os.Remove(clientKeyFile.Name())

	// Write dummy data to the temporary files
	caCertData := []byte(`-----BEGIN CERTIFICATE-----
MIIB3zCCAYWgAwIBAgIUJnstvSViCnkLh+Wmj7+XntqSLckwCgYIKoZIzj0EAwIw
RTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGElu
dGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yNTAxMDgxMzA1NTlaFw0yNzEwMjkx
MzA1NTlaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYD
VQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwWTATBgcqhkjOPQIBBggqhkjO
PQMBBwNCAATyL4bMZn6sFXqb1Pt5VFirRBp7CL0XKoo705ErGypTAMqi9xlAxAJq
D8QjJaLy+Jn3gaqrgoBf0lxFW45Oxw8eo1MwUTAdBgNVHQ4EFgQUyZrX4gaFLC+c
tsQGW94Dt/LQQo0wHwYDVR0jBBgwFoAUyZrX4gaFLC+ctsQGW94Dt/LQQo0wDwYD
VR0TAQH/BAUwAwEB/zAKBggqhkjOPQQDAgNIADBFAiEAv9dmBM5Lot2JWJgehezF
xrBHQq23q9ikSGvtvc5REb8CIF49LbfXowNpfM3sI6d9lyLTswKpnTNBevibjvtr
z0A7
-----END CERTIFICATE-----`)
	if _, err := caCertFile.Write(caCertData); err != nil {
		t.Fatalf("failed to write CA cert data: %v", err)
	}

	clientCertData := []byte(`-----BEGIN CERTIFICATE-----
MIIBzTCCAXSgAwIBAgIUQVPOAfX9jnWKU0iRrtvofKoL47QwCgYIKoZIzj0EAwIw
RTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGElu
dGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yNTAxMDgxMzA2NDhaFw0yNjA1MjMx
MzA2NDhaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYD
VQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwWTATBgcqhkjOPQIBBggqhkjO
PQMBBwNCAARei1eItS/lH7/nlcmFsBi49SmOi/KGGc3CF/aRjmglgl2LVtxPmxYT
huZi+GeY7+GU/AM3TY5JYdibqbTIjNZfo0IwQDAdBgNVHQ4EFgQU2b4MOd0n1dMn
KUou65oDn4h3YVMwHwYDVR0jBBgwFoAUyZrX4gaFLC+ctsQGW94Dt/LQQo0wCgYI
KoZIzj0EAwIDRwAwRAIgQlzLA5Ls9hBSHJchYcDEG5TP/drmKIX4+KTCA8GUhxYC
IDNFATLYp+LzmcjqwYMUtbPoGQmI00H0y2KjtvQN3pau
-----END CERTIFICATE-----`)
	if _, err := clientCertFile.Write(clientCertData); err != nil {
		t.Fatalf("failed to write client cert data: %v", err)
	}

	clientKeyData := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIG8sP1IRZebMoRthB1AmB8ByEje5fi+BEQOrs5DQUezmoAoGCCqGSM49
AwEHoUQDQgAEXotXiLUv5R+/55XJhbAYuPUpjovyhhnNwhf2kY5oJYJdi1bcT5sW
E4bmYvhnmO/hlPwDN02OSWHYm6m0yIzWXw==
-----END EC PRIVATE KEY-----`)
	if _, err := clientKeyFile.Write(clientKeyData); err != nil {
		t.Fatalf("failed to write client key data: %v", err)
	}

	// Test with valid CA cert, client cert, and key
	transport, err := GetTransport(caCertFile.Name(), clientCertFile.Name(), clientKeyFile.Name(), false)
	if err != nil {
		t.Fatalf("GetTransport() returned an error: %v", err)
	}

	// Check if the transport is configured correctly
	if transport.TLSClientConfig == nil {
		t.Fatal("Expected TLSClientConfig to be set")
	}

	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("Expected InsecureSkipVerify to be false")
	}

	// Test with InsecureSkipVerify set to true
	transport, err = GetTransport(caCertFile.Name(), clientCertFile.Name(), clientKeyFile.Name(), true)
	if err != nil {
		t.Fatalf("GetTransport() returned an error: %v", err)
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("Expected InsecureSkipVerify to be true")
	}
}
