package middleware

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/rajweepmondal/url-shortener/pkg/ratelimiter"
)

// RateLimitInterceptor creates a gRPC unary interceptor for rate limiting
func RateLimitInterceptor(middleware *ratelimiter.Middleware) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract IP address from peer
		ip := extractIPFromContext(ctx)
		if ip == "" {
			// If we can't extract IP, allow the request
			return handler(ctx, req)
		}

		// Check rate limit
		allowed, rateLimitInfo, err := middleware.CheckIPRateLimit(ctx, ip)
		if err != nil {
			// Log error but don't fail the request
			// In production, you might want to fail the request or use a fallback
			return handler(ctx, req)
		}

		if !allowed {
			return nil, status.Errorf(codes.ResourceExhausted, 
				"rate limit exceeded: %d requests in window, limit is %d", 
				rateLimitInfo.Count, rateLimitInfo.Limit)
		}

		return handler(ctx, req)
	}
}

// EndpointRateLimitInterceptor creates a rate limiter that limits per endpoint
func EndpointRateLimitInterceptor(middleware *ratelimiter.Middleware) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract IP address from peer
		ip := extractIPFromContext(ctx)
		if ip == "" {
			return handler(ctx, req)
		}

		// Check endpoint-specific rate limit
		allowed, rateLimitInfo, err := middleware.CheckEndpointRateLimit(ctx, info.FullMethod, ip)
		if err != nil {
			return handler(ctx, req)
		}

		if !allowed {
			return nil, status.Errorf(codes.ResourceExhausted, 
				"endpoint rate limit exceeded for %s: %d requests in window, limit is %d", 
				info.FullMethod, rateLimitInfo.Count, rateLimitInfo.Limit)
		}

		return handler(ctx, req)
	}
}

// extractIPFromContext extracts the client IP address from gRPC context
func extractIPFromContext(ctx context.Context) string {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}

	switch addr := peer.Addr.(type) {
	case *net.TCPAddr:
		return addr.IP.String()
	case *net.UDPAddr:
		return addr.IP.String()
	default:
		return ""
	}
}
