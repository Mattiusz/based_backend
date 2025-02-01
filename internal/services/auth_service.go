package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/mattiusz/based_backend/internal/config"
	pb "github.com/mattiusz/based_backend/internal/gen/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService struct {
	config     *config.Config
	httpClient *http.Client
}

func NewAuthService(config *config.Config) (*AuthService, error) {
	return &AuthService{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			// TODO: In production, remove InsecureSkipVerify and configure proper TLS
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}, nil
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.LoginResponse, error) {
	// First create the user
	userData := url.Values{}
	userData.Set("username", req.Username)
	userData.Set("password", req.Password)
	userData.Set("email", req.Email)

	resp, err := s.httpClient.PostForm(s.config.AuthentikURL+"/users/register/", userData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "registration failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, status.Errorf(codes.Internal, "registration failed with status %d", resp.StatusCode)
	}

	// Now get tokens for the new user
	tokenData := url.Values{}
	tokenData.Set("grant_type", "password")
	tokenData.Set("username", req.Username)
	tokenData.Set("password", req.Password)
	tokenData.Set("client_id", s.config.AuthentikClientID)
	tokenData.Set("client_secret", s.config.AuthentikClientSecret)

	tokenResp, err := s.httpClient.PostForm(s.config.AuthentikURL+"/application/o/token/", tokenData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get tokens: %v", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Unauthenticated, "token fetch failed with status %d", tokenResp.StatusCode)
	}

	var tokenResult struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenResult); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decode token response: %v", err)
	}

	return &pb.LoginResponse{
		AccessToken:  tokenResult.AccessToken,
		RefreshToken: tokenResult.RefreshToken,
		ExpiresIn:    tokenResult.ExpiresIn,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	switch auth := req.AuthMethod.(type) {
	case *pb.LoginRequest_Password:
		return s.handlePasswordLogin(ctx, auth.Password)
	case *pb.LoginRequest_Sso:
		return s.handleSSOLogin(ctx, auth.Sso)
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid authentication method")
	}
}

func (s *AuthService) handlePasswordLogin(ctx context.Context, creds *pb.PasswordAuth) (*pb.LoginResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", creds.Username)
	data.Set("password", creds.Password)
	data.Set("client_id", s.config.AuthentikClientID)
	data.Set("client_secret", s.config.AuthentikClientSecret)

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
	data.Set("client_id", s.config.AuthentikClientID)
	data.Set("client_secret", s.config.AuthentikClientSecret)

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

func (s *AuthService) handleSSOLogin(ctx context.Context, sso *pb.SSOAuth) (*pb.LoginResponse, error) {
	// Verify SSO provider is supported
	switch sso.Provider {
	case pb.Provider_GOOGLE, pb.Provider_APPLE:
		// TODO: Implement proper OAuth2 validation
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported SSO provider")
	}

	// In production use proper OAuth2 client verification
	// For now just do basic validation that token exists
	if sso.IdToken == "" {
		return nil, status.Error(codes.InvalidArgument, "missing ID token")
	}

	// Exchange ID token for access token
	data := url.Values{}
	data.Set("grant_type", "id_token")
	data.Set("id_token", sso.IdToken)
	data.Set("client_id", s.config.AuthentikClientID)
	data.Set("client_secret", s.config.AuthentikClientSecret)

	resp, err := s.httpClient.PostForm(s.config.AuthentikURL+"/application/o/token/", data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "SSO login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Unauthenticated, "SSO login failed with status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decode token response: %v", err)
	}

	return &pb.LoginResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}
