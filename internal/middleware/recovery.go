package middleware

import (
	"context"
	"runtime/debug"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryInterceptor creates a gRPC unary interceptor for panic recovery
func RecoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				logger.Error("gRPC handler panicked",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)

				// Convert panic to gRPC error
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor creates a gRPC stream interceptor for panic recovery
func StreamRecoveryInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				logger.Error("gRPC stream handler panicked",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)

				// Convert panic to gRPC error
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, stream)
	}
}

// ValidationInterceptor creates a gRPC unary interceptor for request validation
func ValidationInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Basic validation - check if request is nil
		if req == nil {
			return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
		}

		// You can add more validation logic here
		// For example, validate required fields, format, etc.

		return handler(ctx, req)
	}
}

// MetricsInterceptor creates a gRPC unary interceptor for metrics collection
func MetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// In a real implementation, you would increment metrics here
		// For example, using Prometheus metrics:
		// - Total requests counter
		// - Request duration histogram
		// - Error rate counter
		
		resp, err := handler(ctx, req)

		// Record metrics after request completion
		// This is where you would record:
		// - Response status
		// - Request duration
		// - Method name
		// - Error codes

		return resp, err
	}
}

// CORSInterceptor creates a gRPC unary interceptor for CORS handling
// Note: This is mainly for gRPC-Web, regular gRPC doesn't need CORS
func CORSInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// CORS handling would typically be done at the gateway level
		// for gRPC-Web or HTTP/JSON transcoding
		
		return handler(ctx, req)
	}
}

// TimeoutInterceptor creates a gRPC unary interceptor for request timeouts
func TimeoutInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Check if context already has a deadline
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			// You could set a default timeout here if needed
			// ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			// defer cancel()
		}

		return handler(ctx, req)
	}
}

// AuthInterceptor creates a gRPC unary interceptor for authentication
func AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip authentication for health check and public endpoints
		publicEndpoints := map[string]bool{
			"/url_shortener.v1.URLShortenerService/GetHealthCheck": true,
			"/url_shortener.v1.URLShortenerService/GetOriginalURL": true,
		}

		if publicEndpoints[info.FullMethod] {
			return handler(ctx, req)
		}

		// In a real implementation, you would:
		// 1. Extract authentication token from metadata
		// 2. Validate the token (JWT, API key, etc.)
		// 3. Extract user information
		// 4. Add user info to context
		// 5. Return Unauthenticated error if invalid

		// For now, allow all requests
		return handler(ctx, req)
	}
}

// ChainUnaryInterceptors chains multiple unary interceptors
func ChainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Build the chain of interceptors
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chain
			chain = func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return interceptor(currentCtx, currentReq, info, next)
			}
		}
		return chain(ctx, req)
	}
}
