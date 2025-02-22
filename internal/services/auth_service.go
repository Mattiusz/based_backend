package services

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mattiusz/based_backend/internal/config"
)

type AuthService struct {
	config     *config.Config
	httpClient *http.Client
	publicKeys map[string]interface{}
}

type KeycloakKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type KeycloakKeys struct {
	Keys []KeycloakKey `json:"keys"`
}

// NewAuthService creates a new instance of AuthService
func NewAuthService(config *config.Config) (*AuthService, error) {
	service := &AuthService{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		publicKeys: make(map[string]interface{}),
	}

	// Initialize by fetching public keys
	if err := service.fetchPublicKeys(); err != nil {
		return nil, fmt.Errorf("failed to fetch public keys: %v", err)
	}

	return service, nil
}

// VerifyToken validates the provided JWT token
func (s *AuthService) VerifyToken(ctx context.Context, tokenString string) (*jwt.Token, error) {
	// Remove 'Bearer ' prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from the token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found in token header")
		}

		// Get the public key for this kid
		publicKey, exists := s.publicKeys[kid]
		if !exists {
			// Refresh keys and try again
			if err := s.fetchPublicKeys(); err != nil {
				return nil, fmt.Errorf("failed to refresh public keys: %v", err)
			}

			publicKey, exists = s.publicKeys[kid]
			if !exists {
				return nil, fmt.Errorf("public key not found for kid: %s", kid)
			}
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	// Verify claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Verify audience
		if aud, ok := claims["aud"].(string); ok {
			if aud != s.config.KeycloakClientID {
				return nil, fmt.Errorf("invalid audience")
			}
		}

		// Verify issuer
		expectedIssuer := fmt.Sprintf("http://%s:%s/realms/%s",
			s.config.KeycloakHost,
			s.config.KeycloakPort,
			s.config.KeycloakRealm)
		if iss, ok := claims["iss"].(string); ok {
			if iss != expectedIssuer {
				return nil, fmt.Errorf("invalid issuer")
			}
		}
	}

	return token, nil
}

// fetchPublicKeys retrieves the public keys from Keycloak
func (s *AuthService) fetchPublicKeys() error {
	// Construct the JWKS (JSON Web Key Set) endpoint URL
	jwksURL := fmt.Sprintf("http://%s:%s/realms/%s/protocol/openid-connect/certs",
		s.config.KeycloakHost,
		s.config.KeycloakPort,
		s.config.KeycloakRealm)

	// Make HTTP request to fetch the keys
	resp, err := s.httpClient.Get(jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from JWKS endpoint: %d", resp.StatusCode)
	}

	// Parse the response
	var keycloakKeys KeycloakKeys
	if err := json.NewDecoder(resp.Body).Decode(&keycloakKeys); err != nil {
		return fmt.Errorf("failed to decode JWKS response: %v", err)
	}

	// Clear existing keys
	s.publicKeys = make(map[string]interface{})

	// Parse and store the public keys
	for _, key := range keycloakKeys.Keys {
		if key.Use != "sig" || key.Kty != "RSA" {
			continue // Skip non-RSA signing keys
		}

		// Decode the modulus and exponent
		nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			return fmt.Errorf("failed to decode key modulus: %v", err)
		}

		eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			return fmt.Errorf("failed to decode key exponent: %v", err)
		}

		// Convert the modulus bytes to big.Int
		n := new(big.Int)
		n.SetBytes(nBytes)

		// Convert the exponent bytes to int
		var e int
		for i := 0; i < len(eBytes); i++ {
			e = e<<8 | int(eBytes[i])
		}

		// Create the RSA public key
		publicKey := &rsa.PublicKey{
			N: n,
			E: e,
		}

		// Store the key with its ID
		s.publicKeys[key.Kid] = publicKey
	}

	if len(s.publicKeys) == 0 {
		return fmt.Errorf("no valid RSA signing keys found in JWKS response")
	}

	return nil
}
