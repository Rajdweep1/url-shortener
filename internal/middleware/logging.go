package middleware

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor creates a gRPC unary interceptor for logging
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Extract status code
		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			}
		}

		// Log the request
		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("code", code.String()),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Error("gRPC request failed", fields...)
		} else {
			logger.Info("gRPC request completed", fields...)
		}

		return resp, err
	}
}

// StreamLoggingInterceptor creates a gRPC stream interceptor for logging
func StreamLoggingInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Call the handler
		err := handler(srv, stream)

		// Calculate duration
		duration := time.Since(start)

		// Extract status code
		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			}
		}

		// Log the request
		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("code", code.String()),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Error("gRPC stream failed", fields...)
		} else {
			logger.Info("gRPC stream completed", fields...)
		}

		return err
	}
}
