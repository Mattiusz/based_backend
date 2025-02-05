package interceptors

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthService interface {
	VerifyToken(ctx context.Context, token string) (*jwt.Token, error)
}

// AuthInterceptor handles JWT authentication for gRPC calls
type AuthInterceptor struct {
	authService AuthService
	// Routes that don't need authentication
	publicPaths map[string]bool
}

// NewAuthInterceptor creates a new auth interceptor
func NewAuthInterceptor(authService AuthService) *AuthInterceptor {
	// Initialize paths that don't require authentication
	publicPaths := map[string]bool{
		"/grpc.health.v1.Health/Check": true,
		// Add other public endpoints here
	}

	return &AuthInterceptor{
		authService: authService,
		publicPaths: publicPaths,
	}
}

// Unary returns a server interceptor function to handle authentication for unary RPC
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Check if the path requires authentication
		if i.publicPaths[info.FullMethod] {
			return handler(ctx, req)
		}

		// Extract token from metadata
		token, err := i.extractToken(ctx)
		if err != nil {
			return nil, err
		}

		// Verify token
		claims, err := i.authService.VerifyToken(ctx, token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Add claims to context
		newCtx := context.WithValue(ctx, "claims", claims)

		return handler(newCtx, req)
	}
}

// Stream returns a server interceptor function to handle authentication for stream RPC
func (i *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Check if the path requires authentication
		if i.publicPaths[info.FullMethod] {
			return handler(srv, stream)
		}

		// Extract token from metadata
		token, err := i.extractToken(stream.Context())
		if err != nil {
			return err
		}

		// Verify token
		claims, err := i.authService.VerifyToken(stream.Context(), token)
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Wrap the stream with the authenticated context
		wrappedStream := &wrappedServerStream{
			ServerStream: stream,
			ctx:          context.WithValue(stream.Context(), "claims", claims),
		}

		return handler(srv, wrappedStream)
	}
}

// extractToken gets the token from the gRPC metadata
func (i *AuthInterceptor) extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	token := values[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return "", status.Errorf(codes.Unauthenticated, "invalid authorization format")
	}

	return strings.TrimPrefix(token, "Bearer "), nil
}

// wrappedServerStream wraps grpc.ServerStream to modify the context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
