package httpapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var keyIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,128}$`)

type albClaimsVerifier struct {
	expectedSigner string
	region         string
	client         *http.Client
	mu             sync.RWMutex
	keys           map[string]*ecdsa.PublicKey
}

type albJWTHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Signer    string `json:"signer"`
	ExpiresAt int64  `json:"exp"`
}

type albClaims struct {
	Subject string `json:"sub"`
}

func newALBClaimsVerifier(expectedSigner, region string) *albClaimsVerifier {
	return &albClaimsVerifier{expectedSigner: expectedSigner, region: region, client: &http.Client{Timeout: 3 * time.Second}, keys: map[string]*ecdsa.PublicKey{}}
}

func (v *albClaimsVerifier) Verify(ctx context.Context, token string) (string, error) {
	if v.expectedSigner == "" {
		return "", fmt.Errorf("expected signer is not configured")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("signed claims token is malformed")
	}
	var header albJWTHeader
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return "", fmt.Errorf("decode signed claims header: %w", err)
	}
	if header.Algorithm != "ES256" || header.Signer != v.expectedSigner || header.ExpiresAt <= time.Now().Unix() || !keyIDPattern.MatchString(header.KeyID) {
		return "", fmt.Errorf("signed claims header is not trusted")
	}
	key, err := v.publicKey(ctx, header.KeyID)
	if err != nil {
		return "", err
	}
	signature, err := decodeBase64URL(parts[2])
	if err != nil || len(signature) != 64 {
		return "", fmt.Errorf("signed claims signature is malformed")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(key, digest[:], new(big.Int).SetBytes(signature[:32]), new(big.Int).SetBytes(signature[32:])) {
		return "", fmt.Errorf("signed claims signature is invalid")
	}
	var claims albClaims
	if err := decodeJWTPart(parts[1], &claims); err != nil || claims.Subject == "" {
		return "", fmt.Errorf("signed claims subject is missing")
	}
	return claims.Subject, nil
}

func (v *albClaimsVerifier) publicKey(ctx context.Context, keyID string) (*ecdsa.PublicKey, error) {
	v.mu.RLock()
	key := v.keys[keyID]
	v.mu.RUnlock()
	if key != nil {
		return key, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://public-keys.auth.elb."+v.region+".amazonaws.com/"+keyID, nil)
	if err != nil {
		return nil, err
	}
	response, err := v.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("retrieve ALB signing key: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("retrieve ALB signing key: HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16<<10))
	if err != nil {
		return nil, fmt.Errorf("read ALB signing key: %w", err)
	}
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, fmt.Errorf("ALB signing key is not PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse ALB signing key: %w", err)
	}
	key, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("ALB signing key is not ECDSA")
	}
	v.mu.Lock()
	v.keys[keyID] = key
	v.mu.Unlock()
	return key, nil
}

func decodeJWTPart(value string, target any) error {
	decoded, err := decodeBase64URL(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(decoded, target)
}

func decodeBase64URL(value string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	return base64.URLEncoding.DecodeString(value)
}
