package checks

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/pgdepaula/vyst-openauth/api/proto"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/config"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/runner"
)

// GRPCChecks returns gRPC API verification checks.
// These checks focus on S2S operations: token validation and RBAC queries.
func GRPCChecks(authCtx *AuthContext) []runner.Check {
	return []runner.Check{
		{
			Name:  "gRPC Connectivity",
			Group: "gRPC",
			Fn:    makeGRPCConnectivityCheck(),
		},
		{
			Name:  "gRPC ValidateToken",
			Group: "gRPC",
			Fn:    makeGRPCValidateTokenCheck(authCtx),
		},
		{
			Name:  "gRPC GetUserRoles",
			Group: "gRPC",
			Fn:    makeGRPCGetUserRolesCheck(authCtx),
		},
		{
			Name:  "gRPC ValidateCompanyAccess",
			Group: "gRPC",
			Fn:    makeGRPCValidateCompanyAccessCheck(authCtx),
		},
	}
}

func makeGRPCValidateCompanyAccessCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.CompanyID == "" {
			return &runner.CheckResult{
				Name:     "gRPC ValidateCompanyAccess",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no company_id available",
			}, nil
		}

		conn, err := newGRPCClientConn(cfg.GRPCURL)
		if err != nil {
			return &runner.CheckResult{
				Name:     "gRPC ValidateCompanyAccess",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    fmt.Sprintf("failed to connect: %v", err),
			}, nil
		}
		defer func() { _ = conn.Close() }()

		client := pb.NewIdentityServiceClient(conn)

		resp, err := client.ValidateCompanyAccess(ctx, &pb.ValidateCompanyAccessRequest{
			Token:     authCtx.Token,
			CompanyId: authCtx.CompanyID,
		})
		if err != nil {
			return &runner.CheckResult{
				Name:     "gRPC ValidateCompanyAccess",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if !resp.Valid {
			return &runner.CheckResult{
				Name:     "gRPC ValidateCompanyAccess",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "access denied",
			}, nil
		}

		return &runner.CheckResult{
			Name:     "gRPC ValidateCompanyAccess",
			Group:    "gRPC",
			Passed:   true,
			Duration: time.Since(start),
		}, nil
	}
}

func makeGRPCConnectivityCheck() func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		conn, err := newGRPCClientConn(cfg.GRPCURL)
		if err != nil {
			return &runner.CheckResult{
				Name:     "gRPC Connectivity",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    fmt.Sprintf("failed to connect: %v", err),
			}, nil
		}
		defer func() { _ = conn.Close() }()

		if err := waitForGRPCReady(ctx, conn, cfg.Timeout); err != nil {
			return &runner.CheckResult{
				Name:     "gRPC Connectivity",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    fmt.Sprintf("failed to connect: %v", err),
			}, nil
		}

		return &runner.CheckResult{
			Name:     "gRPC Connectivity",
			Group:    "gRPC",
			Passed:   true,
			Duration: time.Since(start),
		}, nil
	}
}

// makeGRPCValidateTokenCheck tests the ValidateToken S2S method.
// Uses the token obtained from REST authentication flow.
func makeGRPCValidateTokenCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		// Need a token from REST auth flow
		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "gRPC ValidateToken",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available (REST auth flow must run first)",
			}, nil
		}

		conn, err := newGRPCClientConn(cfg.GRPCURL)
		if err != nil {
			return &runner.CheckResult{
				Name:     "gRPC ValidateToken",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    fmt.Sprintf("failed to connect: %v", err),
			}, nil
		}
		defer func() { _ = conn.Close() }()

		client := pb.NewIdentityServiceClient(conn)

		resp, err := client.ValidateToken(ctx, &pb.ValidateTokenRequest{
			Token: authCtx.Token,
		})
		if err != nil {
			return &runner.CheckResult{
				Name:     "gRPC ValidateToken",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if !resp.Valid {
			return &runner.CheckResult{
				Name:     "gRPC ValidateToken",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "token validation returned invalid",
			}, nil
		}

		if resp.UserId == "" {
			return &runner.CheckResult{
				Name:     "gRPC ValidateToken",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "received empty user_id",
			}, nil
		}

		return &runner.CheckResult{
			Name:     "gRPC ValidateToken",
			Group:    "gRPC",
			Passed:   true,
			Duration: time.Since(start),
		}, nil
	}
}

// makeGRPCGetUserRolesCheck tests the GetUserRoles S2S method.
func makeGRPCGetUserRolesCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.UserID == "" {
			return &runner.CheckResult{
				Name:     "gRPC GetUserRoles",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no user_id available from auth flow",
			}, nil
		}

		conn, err := newGRPCClientConn(cfg.GRPCURL)
		if err != nil {
			return &runner.CheckResult{
				Name:     "gRPC GetUserRoles",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    fmt.Sprintf("failed to connect: %v", err),
			}, nil
		}
		defer func() { _ = conn.Close() }()

		client := pb.NewIdentityServiceClient(conn)

		resp, err := client.GetUserRoles(ctx, &pb.GetUserRolesRequest{
			UserId:   authCtx.UserID,
			TenantId: authCtx.TenantID,
		})
		if err != nil {
			return &runner.CheckResult{
				Name:     "gRPC GetUserRoles",
				Group:    "gRPC",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		// Roles can be empty, that's OK - we just check the call succeeds
		_ = resp.Roles

		return &runner.CheckResult{
			Name:     "gRPC GetUserRoles",
			Group:    "gRPC",
			Passed:   true,
			Duration: time.Since(start),
		}, nil
	}
}

func newGRPCClientConn(target string) (*grpc.ClientConn, error) {
	return grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func waitForGRPCReady(ctx context.Context, conn *grpc.ClientConn, timeout time.Duration) error {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if !conn.WaitForStateChange(waitCtx, state) {
			return waitCtx.Err()
		}
	}
}
