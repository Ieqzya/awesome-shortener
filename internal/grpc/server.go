// Package grpc содержит gRPC сервер для сервиса сокращения URL.
package grpc

import (
	"context"
	"strings"

	"awesome-shortener/internal/auth"
	"awesome-shortener/internal/config"
	pb "awesome-shortener/internal/grpc/pb"
	"awesome-shortener/internal/service"
	"awesome-shortener/internal/storage"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ShortenerServer реализует gRPC сервер для сокращения URL
type ShortenerServer struct {
	pb.UnimplementedShortenerServiceServer
	config      *config.Config
	urlService  *service.URLService
	authService *auth.AuthService
}

// NewShortenerServer создает новый gRPC сервер
func NewShortenerServer(cfg *config.Config, urlSvc *service.URLService, authSvc *auth.AuthService) *ShortenerServer {
	return &ShortenerServer{
		config:      cfg,
		urlService:  urlSvc,
		authService: authSvc,
	}
}

// getUserIDFromContext извлекает userID из metadata
func (s *ShortenerServer) getUserIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// Если metadata нет, создаем нового пользователя
		return s.authService.GenerateUserID()
	}

	// Проверяем authorization header
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		// Если authorization нет, создаем нового пользователя
		return s.authService.GenerateUserID()
	}

	// Извлекаем userID из authorization header
	authHeader := authHeaders[0]
	
	// Формат: "Bearer <signed_user_id>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return s.authService.GenerateUserID()
	}

	signedUserID := parts[1]
	
	// Проверяем подпись и извлекаем userID
	userID, valid := s.authService.VerifySignedValue(signedUserID)
	if !valid {
		// Если подпись невалидна, создаем нового пользователя
		return s.authService.GenerateUserID()
	}

	return userID
}

// setUserIDToContext добавляет userID в outgoing metadata
func (s *ShortenerServer) setUserIDToContext(ctx context.Context, userID string) context.Context {
	// Подписываем userID
	signedUserID := s.authService.SignValue(userID)

	// Добавляем в outgoing metadata
	md := metadata.Pairs("authorization", "Bearer "+signedUserID)
	return metadata.NewOutgoingContext(ctx, md)
}

// ShortenURL сокращает URL
func (s *ShortenerServer) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	if req.Url == "" {
		return nil, status.Errorf(codes.InvalidArgument, "url cannot be empty")
	}

	// Получаем или создаем userID
	userID := s.getUserIDFromContext(ctx)

	// Сокращаем URL через сервис
	shortURL, statusCode, err := s.urlService.ShortenURL(ctx, req.Url, userID)
	if err != nil && statusCode >= 500 {
		return nil, status.Errorf(codes.Internal, "failed to shorten URL: %v", err)
	}

	// Добавляем userID в ответ
	ctx = s.setUserIDToContext(ctx, userID)
	
	// Отправляем header с authorization
	signedUserID := s.authService.SignValue(userID)
	if err := grpc.SendHeader(ctx, metadata.Pairs("authorization", "Bearer "+signedUserID)); err != nil {
		// Игнорируем ошибку отправки header
	}

	return &pb.URLShortenResponse{
		Result: shortURL,
	}, nil
}

// ExpandURL получает оригинальный URL по короткому ID
func (s *ShortenerServer) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	if req.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id cannot be empty")
	}

	// Получаем оригинальный URL
	originalURL, err := s.urlService.GetOriginalURL(ctx, req.Id)
	if err != nil {
		if storage.IsDeletedError(err) {
			return nil, status.Errorf(codes.NotFound, "URL has been deleted")
		}
		return nil, status.Errorf(codes.NotFound, "URL not found")
	}

	return &pb.URLExpandResponse{
		Result: originalURL,
	}, nil
}

// ListUserURLs возвращает все URL пользователя
func (s *ShortenerServer) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	// Получаем userID
	userID := s.getUserIDFromContext(ctx)

	// Получаем URL пользователя
	records, err := s.urlService.GetUserURLs(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user URLs: %v", err)
	}

	// Конвертируем в protobuf формат
	urls := make([]*pb.URLData, 0, len(records))
	for _, record := range records {
		urls = append(urls, &pb.URLData{
			ShortUrl:    record.ShortURL,
			OriginalUrl: record.OriginalURL,
		})
	}

	return &pb.UserURLsResponse{
		Url: urls,
	}, nil
}
