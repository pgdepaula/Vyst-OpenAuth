// Package grpc provides the gRPC server implementation.
//
// This gRPC API is designed for high-performance Service-to-Service (S2S)
// authorization operations. Authentication (Login, Register, etc.) is
// handled by the REST API.
package grpc

import (
	"context"
	"sync"
	"time"

	"errors"

	pb "github.com/pgdepaula/vyst-openauth/api/proto"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the IdentityService gRPC server.
// Optimized for S2S authorization and session management.
type Server struct {
	pb.UnimplementedIdentityServiceServer
	tokenSvc        ports.TokenService
	policyRepo      policy.Repository
	companyUserRepo company.CompanyUserRepository
	logger          ports.Logger

	// Active streams for Kill Switch session monitoring
	streamsMu sync.RWMutex
	streams   map[string][]chan struct{}
}

func safeInt32(value int) int32 {
	const (
		maxInt32 = 1<<31 - 1
		minInt32 = -1 << 31
	)

	if value > maxInt32 {
		return maxInt32
	}
	if value < minInt32 {
		return minInt32
	}
	return int32(value)
}

// NewServer creates a new gRPC server for S2S operations.
func NewServer(
	tokenSvc ports.TokenService,
	policyRepo policy.Repository,
	companyUserRepo company.CompanyUserRepository,
	logger ports.Logger,
) *Server {
	return &Server{
		tokenSvc:        tokenSvc,
		policyRepo:      policyRepo,
		companyUserRepo: companyUserRepo,
		logger:          logger,
		streams:         make(map[string][]chan struct{}),
	}
}

// ValidateToken validates a JWT access token.
// Optimized for stateless RSA signature verification.
// This is the primary S2S authentication method.
func (s *Server) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &pb.ValidateTokenResponse{Valid: false}, nil
	}

	claims, err := s.tokenSvc.ValidateToken(req.Token)
	if err != nil {
		s.logger.Debug("Token validation failed", "error", err)
		return &pb.ValidateTokenResponse{Valid: false}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:    true,
		UserId:   claims.UserID,
		TenantId: claims.TenantID,
		Roles:    claims.Roles,
	}, nil
}

// StreamValidateSession implements the Kill Switch streaming RPC.
// Clients can subscribe to session status updates for real-time revocation.
func (s *Server) StreamValidateSession(stream pb.IdentityService_StreamValidateSessionServer) error {
	// 1. Receive initial request with session identifier
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	tokenOrSessionID := req.SessionId
	if tokenOrSessionID == "" {
		return status.Error(codes.InvalidArgument, "session_id (token) is required")
	}

	// Validate token to get UserID for Kill Switch tracking
	claims, err := s.tokenSvc.ValidateToken(tokenOrSessionID)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}
	userID := claims.UserID

	s.logger.Info("Session stream started", "user_id", userID)

	// 2. Register stream for Kill Switch notifications
	killChan := make(chan struct{})
	s.streamsMu.Lock()
	s.streams[userID] = append(s.streams[userID], killChan)
	s.streamsMu.Unlock()

	defer func() {
		s.streamsMu.Lock()
		channels := s.streams[userID]
		for i, ch := range channels {
			if ch == killChan {
				s.streams[userID] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		if len(s.streams[userID]) == 0 {
			delete(s.streams, userID)
		}
		s.streamsMu.Unlock()
		s.logger.Info("Session stream closed", "user_id", userID)
	}()

	// 3. Send initial "Active" response
	if err := stream.Send(&pb.ValidateSessionResponse{Active: true}); err != nil {
		return err
	}

	// 4. Keep stream alive with periodic heartbeats
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-killChan:
			s.logger.Info("Kill Switch triggered for stream", "user_id", userID)
			return stream.Send(&pb.ValidateSessionResponse{
				Active: false,
				Reason: "Session terminated by security policy",
			})
		case <-time.After(30 * time.Second):
			// Heartbeat to keep connection alive
			if err := stream.Send(&pb.ValidateSessionResponse{Active: true}); err != nil {
				return err
			}
		}
	}
}

// GetUserRoles returns the roles assigned to a user.
// Used by backend services for authorization decisions.
func (s *Server) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	roles, err := s.policyRepo.GetRolesForUser(ctx, req.UserId)
	if err != nil {
		s.logger.Error("Failed to get roles for user", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get roles: %v", err)
	}

	return &pb.GetUserRolesResponse{
		Roles: roles,
	}, nil
}

// GetUserPermissions returns the full permission set for a user.
// Aggregates permissions from all user roles.
func (s *Server) GetUserPermissions(ctx context.Context, req *pb.GetUserPermissionsRequest) (*pb.GetUserPermissionsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// First get user roles
	roles, err := s.policyRepo.GetRolesForUser(ctx, req.UserId)
	if err != nil {
		s.logger.Error("Failed to get roles for user", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get roles: %v", err)
	}

	// Aggregate permissions from all roles
	// For now, we return role names as permissions until we have granular permission system
	permissions := make([]*pb.Permission, 0, len(roles))
	for _, role := range roles {
		permissions = append(permissions, &pb.Permission{
			Resource:   "*",
			Action:     role,
			ResourceId: "*",
		})
	}

	return &pb.GetUserPermissionsResponse{
		Permissions: permissions,
	}, nil
}

// RevokeUserSessions terminates all active sessions for a user.
// This triggers the Kill Switch mechanism across all connected services.
func (s *Server) RevokeUserSessions(ctx context.Context, req *pb.RevokeUserSessionsRequest) (*pb.RevokeUserSessionsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	s.logger.Warn("Revoking all sessions for user",
		"user_id", req.UserId,
		"reason", req.Reason,
	)

	// Trigger Kill Switch for all active streams
	revokedCount := s.triggerKillSwitch(req.UserId)

	s.logger.Info("Sessions revoked",
		"user_id", req.UserId,
		"revoked_count", revokedCount,
	)

	return &pb.RevokeUserSessionsResponse{
		RevokedCount: safeInt32(revokedCount),
		Success:      true,
	}, nil
}

// triggerKillSwitch terminates all active session streams for a user.
// Returns the number of sessions that were revoked.
func (s *Server) triggerKillSwitch(userID string) int {
	s.streamsMu.RLock()
	channels, ok := s.streams[userID]
	s.streamsMu.RUnlock()

	if !ok {
		return 0
	}

	count := 0
	for _, ch := range channels {
		select {
		case <-ch:
			// Already closed
		default:
			close(ch)
			count++
		}
	}

	return count
}

// TriggerKillSwitch is exported for use by other components (e.g., HTTP handlers).
// Terminates all sessions for a user.
func (s *Server) TriggerKillSwitch(userID string) {
	s.triggerKillSwitch(userID)
}

// ValidateCompanyAccess validates if a token holder has access to a specific company.
// Returns the user's role in that company if allowed.
func (s *Server) ValidateCompanyAccess(ctx context.Context, req *pb.ValidateCompanyAccessRequest) (*pb.ValidateCompanyAccessResponse, error) {
	if req.Token == "" || req.CompanyId == "" {
		return &pb.ValidateCompanyAccessResponse{Valid: false}, nil
	}

	// 1. Validate Token
	claims, err := s.tokenSvc.ValidateToken(req.Token)
	if err != nil {
		s.logger.Debug("Token validation failed inside ValidateCompanyAccess", "error", err)
		return &pb.ValidateCompanyAccessResponse{Valid: false}, nil
	}

	// 2. Check Company Access
	role, err := s.companyUserRepo.GetUserRole(ctx, req.CompanyId, claims.UserID)
	if err != nil {
		if errors.Is(err, company.ErrUserNotMember) {
			s.logger.Info("Access denied: User not member of company", "user_id", claims.UserID, "company_id", req.CompanyId)
			return &pb.ValidateCompanyAccessResponse{Valid: false}, nil
		}
		s.logger.Error("Failed to check company access", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to check company access: %v", err)
	}

	return &pb.ValidateCompanyAccessResponse{
		Valid:  true,
		UserId: claims.UserID,
		Role:   string(role),
	}, nil
}

// GetCompanyRoles returns all company memberships for a user.
func (s *Server) GetCompanyRoles(ctx context.Context, req *pb.GetCompanyRolesRequest) (*pb.GetCompanyRolesResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	companies, err := s.companyUserRepo.GetCompaniesForUser(ctx, req.UserId)
	if err != nil {
		s.logger.Error("Failed to get user companies", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get user companies: %v", err)
	}

	roles := make([]*pb.CompanyRoleEntry, 0, len(companies))
	for _, c := range companies {
		roles = append(roles, &pb.CompanyRoleEntry{
			CompanyId: c.CompanyID,
			Role:      string(c.Role),
		})
	}

	return &pb.GetCompanyRolesResponse{Roles: roles}, nil
}
