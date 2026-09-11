package httpapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestALBClaimsVerifier(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded})
	signer := "arn:aws:elasticloadbalancing:us-west-2:123456789012:loadbalancer/app/kubevista/test"
	verifier := newALBClaimsVerifier(signer, "us-west-2")
	verifier.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(request.URL.Path, "/test-key") {
			t.Fatalf("unexpected key URL: %s", request.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(publicKey)))}, nil
	})}
	token := signedALBToken(t, key, signer, "operator-123")
	subject, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if subject != "operator-123" {
		t.Fatalf("unexpected subject %q", subject)
	}
	if _, err := newALBClaimsVerifier(signer+"-wrong", "us-west-2").Verify(context.Background(), token); err == nil {
		t.Fatal("expected signer mismatch")
	}
}

func signedALBToken(t *testing.T, key *ecdsa.PrivateKey, signer, subject string) string {
	t.Helper()
	header, _ := json.Marshal(map[string]any{"alg": "ES256", "kid": "test-key", "signer": signer, "exp": time.Now().Add(time.Minute).Unix()})
	payload, _ := json.Marshal(map[string]string{"sub": subject})
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(unsigned))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature := append(paddedCoordinate(r.Bytes()), paddedCoordinate(s.Bytes())...)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func paddedCoordinate(value []byte) []byte {
	result := make([]byte, 32)
	copy(result[32-len(value):], value)
	return result
}
