package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	pb "github.com/mattiusz/based_backend/internal/gen/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Config struct {
	ClientID     string
	ClientSecret string
	AuthentikURL string
}

type AuthService struct {
	config     Config
	httpClient *http.Client
}

func NewAuthService() (*AuthService, error) {
	clientID := os.Getenv("AUTHENTIK_CLIENT_ID")
	clientSecret := os.Getenv("AUTHENTIK_CLIENT_SECRET")
	authentikURL := os.Getenv("AUTHENTIK_URL")
	if clientID == "" || clientSecret == "" || authentikURL == "" {
		return nil, fmt.Errorf("missing authentik configuration")
	}

	return &AuthService{
		config: Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			AuthentikURL: authentikURL,
		},
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			// TODO: In production, remove InsecureSkipVerify and configure proper TLS
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	switch auth := req.AuthMethod.(type) {
	case *pb.LoginRequest_Password:
		return s.handlePasswordLogin(ctx, auth.Password)
	case *pb.LoginRequest_Sso:
		return nil, status.Error(codes.Unimplemented, "SSO login not implemented")
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid authentication method")
	}
}

func (s *AuthService) handlePasswordLogin(ctx context.Context, creds *pb.PasswordAuth) (*pb.LoginResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", creds.Username)
	data.Set("password", creds.Password)
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)

	resp, err := s.httpClient.PostForm(s.config.AuthentikURL+"/application/o/token/", data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to authenticate: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Unauthenticated, "authentication failed with status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decode response: %v", err)
	}

	return &pb.LoginResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	data := url.Values{}
	data.Set("token", req.AccessToken)
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)

	resp, err := s.httpClient.PostForm(s.config.AuthentikURL+"/application/o/introspect/", data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Unauthenticated, "token validation failed with status %d", resp.StatusCode)
	}

	var introspectResp struct {
		Active bool     `json:"active"`
		Claims []string `json:"claims"`
		Sub    string   `json:"sub"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&introspectResp); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decode introspection response: %v", err)
	}

	return &pb.ValidateTokenResponse{
		Valid:  introspectResp.Active,
		Claims: introspectResp.Claims,
		UserId: introspectResp.Sub,
	}, nil
}
