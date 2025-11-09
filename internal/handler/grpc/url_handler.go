package grpc

import (
	"context"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rajweepmondal/url-shortener/internal/models"
	"github.com/rajweepmondal/url-shortener/internal/service"
	pb "github.com/rajweepmondal/url-shortener/proto/gen/go/url_shortener/v1"
)

// URLHandler implements the gRPC URL shortener service
type URLHandler struct {
	pb.UnimplementedURLShortenerServiceServer
	urlService *service.URLService
}

// NewURLHandler creates a new URL handler
func NewURLHandler(urlService *service.URLService) *URLHandler {
	return &URLHandler{
		urlService: urlService,
	}
}

// ShortenURL creates a new shortened URL
func (h *URLHandler) ShortenURL(ctx context.Context, req *pb.ShortenURLRequest) (*pb.ShortenURLResponse, error) {
	// Convert protobuf request to domain model
	createReq := &models.CreateURLRequest{
		OriginalURL: req.OriginalUrl,
		UserID:      req.UserId,
	}

	if req.CustomAlias != nil {
		createReq.CustomAlias = req.CustomAlias
	}

	if req.ExpiresIn != nil {
		duration := req.ExpiresIn.AsDuration()
		createReq.ExpiresIn = &duration
	}

	// Call service
	url, shortURL, err := h.urlService.ShortenURL(ctx, createReq)
	if err != nil {
		return nil, h.handleError(err)
	}

	// Convert domain model to protobuf response
	return &pb.ShortenURLResponse{
		Url:      h.urlToProto(url),
		ShortUrl: shortURL,
	}, nil
}

// GetOriginalURL retrieves the original URL for redirection
func (h *URLHandler) GetOriginalURL(ctx context.Context, req *pb.GetOriginalURLRequest) (*pb.GetOriginalURLResponse, error) {
	// Extract client information from context
	clientInfo := h.extractClientInfo(ctx, req)

	// Call service
	originalURL, err := h.urlService.GetOriginalURL(ctx, req.ShortCode, clientInfo)
	if err != nil {
		return nil, h.handleError(err)
	}

	return &pb.GetOriginalURLResponse{
		OriginalUrl: originalURL,
		IsExpired:   false,
		IsActive:    true,
	}, nil
}

// GetURLInfo retrieves detailed information about a URL
func (h *URLHandler) GetURLInfo(ctx context.Context, req *pb.GetURLInfoRequest) (*pb.GetURLInfoResponse, error) {
	url, err := h.urlService.GetURLInfo(ctx, req.ShortCode, req.UserId)
	if err != nil {
		return nil, h.handleError(err)
	}

	return &pb.GetURLInfoResponse{
		Url: h.urlToProto(url),
	}, nil
}

// ListURLs retrieves a paginated list of URLs
func (h *URLHandler) ListURLs(ctx context.Context, req *pb.ListURLsRequest) (*pb.ListURLsResponse, error) {
	// Convert protobuf request to domain model
	listReq := &models.ListURLsRequest{
		UserID:   req.UserId,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
	}

	// Call service
	response, err := h.urlService.ListURLs(ctx, listReq)
	if err != nil {
		return nil, h.handleError(err)
	}

	// Convert domain model to protobuf response
	urls := make([]*pb.URL, len(response.URLs))
	for i, url := range response.URLs {
		urls[i] = h.urlToProto(url)
	}

	return &pb.ListURLsResponse{
		Urls:       urls,
		TotalCount: response.TotalCount,
		TotalPages: int32(response.TotalPages),
		Page:       int32(response.Page),
		PageSize:   int32(response.PageSize),
	}, nil
}

// UpdateURL updates an existing URL
func (h *URLHandler) UpdateURL(ctx context.Context, req *pb.UpdateURLRequest) (*pb.UpdateURLResponse, error) {
	// Convert protobuf request to domain model
	updates := &models.URL{
		CustomAlias: req.CustomAlias,
		IsActive:    req.IsActive != nil && *req.IsActive,
	}

	if req.OriginalUrl != nil {
		updates.OriginalURL = *req.OriginalUrl
	}

	if req.ExpiresAt != nil {
		expiresAt := req.ExpiresAt.AsTime()
		updates.ExpiresAt = &expiresAt
	}

	// Call service
	url, err := h.urlService.UpdateURL(ctx, req.ShortCode, updates, req.UserId)
	if err != nil {
		return nil, h.handleError(err)
	}

	return &pb.UpdateURLResponse{
		Url: h.urlToProto(url),
	}, nil
}

// DeleteURL soft deletes a URL
func (h *URLHandler) DeleteURL(ctx context.Context, req *pb.DeleteURLRequest) (*pb.DeleteURLResponse, error) {
	err := h.urlService.DeleteURL(ctx, req.ShortCode, req.UserId)
	if err != nil {
		return nil, h.handleError(err)
	}

	return &pb.DeleteURLResponse{
		Success: true,
		Message: "URL deleted successfully",
	}, nil
}

// GetAnalytics retrieves analytics data for a URL
func (h *URLHandler) GetAnalytics(ctx context.Context, req *pb.GetAnalyticsRequest) (*pb.GetAnalyticsResponse, error) {
	// Set default time range if not provided
	from := time.Now().AddDate(0, 0, -30) // 30 days ago
	to := time.Now()

	if req.FromDate != nil {
		from = req.FromDate.AsTime()
	}
	if req.ToDate != nil {
		to = req.ToDate.AsTime()
	}

	// Call service
	stats, err := h.urlService.GetAnalytics(ctx, req.ShortCode, from, to, req.UserId)
	if err != nil {
		return nil, h.handleError(err)
	}

	// Convert to protobuf response
	return &pb.GetAnalyticsResponse{
		Stats: h.urlStatsToProto(stats),
	}, nil
}

// GetHealthCheck provides service health information
func (h *URLHandler) GetHealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:    "healthy",
		Timestamp: timestamppb.Now(),
		Version:   "1.0.0",
		Dependencies: map[string]string{
			"database": "healthy",
			"cache":    "healthy",
		},
	}, nil
}

// Helper methods

// urlToProto converts a domain URL model to protobuf
func (h *URLHandler) urlToProto(url *models.URL) *pb.URL {
	pbURL := &pb.URL{
		Id:          url.ID.String(),
		ShortCode:   url.ShortCode,
		OriginalUrl: url.OriginalURL,
		CreatedAt:   timestamppb.New(url.CreatedAt),
		UpdatedAt:   timestamppb.New(url.UpdatedAt),
		ClickCount:  url.ClickCount,
		IsActive:    url.IsActive,
	}

	if url.LastAccessedAt != nil {
		pbURL.LastAccessedAt = timestamppb.New(*url.LastAccessedAt)
	}

	if url.CustomAlias != nil {
		pbURL.CustomAlias = url.CustomAlias
	}

	if url.UserID != nil {
		pbURL.UserId = url.UserID
	}

	if url.ExpiresAt != nil {
		pbURL.ExpiresAt = timestamppb.New(*url.ExpiresAt)
	}

	return pbURL
}

// urlStatsToProto converts URL stats to protobuf
func (h *URLHandler) urlStatsToProto(stats *models.URLStats) *pb.URLStats {
	return &pb.URLStats{
		Url:            h.urlToProto(stats.URL),
		TotalClicks:    stats.TotalClicks,
		UniqueClicks:   stats.UniqueClicks,
		ClicksToday:    stats.ClicksToday,
		ClicksThisWeek: stats.ClicksThisWeek,
		TopCountries:   stats.TopCountries,
		TopReferers:    stats.TopReferers,
	}
}

// extractClientInfo extracts client information from gRPC context
func (h *URLHandler) extractClientInfo(ctx context.Context, req *pb.GetOriginalURLRequest) *service.ClientInfo {
	clientInfo := &service.ClientInfo{}

	// Extract IP address from request or peer info
	if req.IpAddress != nil {
		clientInfo.IPAddress = req.IpAddress
	} else if peerInfo, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := peerInfo.Addr.(*net.TCPAddr); ok {
			ip := tcpAddr.IP.String()
			clientInfo.IPAddress = &ip
		}
	}

	// Extract user agent from request or metadata
	if req.UserAgent != nil {
		clientInfo.UserAgent = req.UserAgent
	} else if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userAgents := md.Get("user-agent"); len(userAgents) > 0 {
			clientInfo.UserAgent = &userAgents[0]
		}
	}

	// Extract referer from request or metadata
	if req.Referer != nil {
		clientInfo.Referer = req.Referer
	} else if md, ok := metadata.FromIncomingContext(ctx); ok {
		if referers := md.Get("referer"); len(referers) > 0 {
			clientInfo.Referer = &referers[0]
		}
	}

	return clientInfo
}

// handleError converts domain errors to gRPC errors
func (h *URLHandler) handleError(err error) error {
	if appErr, ok := err.(*models.AppError); ok {
		switch appErr.Code {
		case models.ErrCodeBadRequest:
			return status.Error(codes.InvalidArgument, appErr.Message)
		case models.ErrCodeUnauthorized:
			return status.Error(codes.Unauthenticated, appErr.Message)
		case models.ErrCodeForbidden:
			return status.Error(codes.PermissionDenied, appErr.Message)
		case models.ErrCodeNotFound:
			return status.Error(codes.NotFound, appErr.Message)
		case models.ErrCodeConflict:
			return status.Error(codes.AlreadyExists, appErr.Message)
		case models.ErrCodeValidation:
			return status.Error(codes.InvalidArgument, appErr.Message)
		case models.ErrCodeRateLimit:
			return status.Error(codes.ResourceExhausted, appErr.Message)
		case models.ErrCodeInternal:
			return status.Error(codes.Internal, appErr.Message)
		case models.ErrCodeDatabase:
			return status.Error(codes.Internal, "database error")
		case models.ErrCodeCache:
			return status.Error(codes.Internal, "cache error")
		case models.ErrCodeExternal:
			return status.Error(codes.Unavailable, appErr.Message)
		default:
			return status.Error(codes.Internal, "internal server error")
		}
	}

	// Handle standard errors
	if strings.Contains(err.Error(), "not found") {
		return status.Error(codes.NotFound, err.Error())
	}

	return status.Error(codes.Internal, "internal server error")
}
