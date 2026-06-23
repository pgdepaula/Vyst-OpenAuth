// Package sdk provides a Go client for the Vyst Identity API.
//
// The SDK simplifies integration with Vyst Identity by handling:
//   - Authentication (login, logout)
//   - Automatic token refresh
//   - Permission checks via ReBAC
//   - Making authenticated API requests
//
// # Quick Start
//
//	import "github.com/pgdepaula/vyst-openauth/pkg/sdk"
//
//	// Create client
//	client := sdk.NewClient("https://auth.vyst.com.br")
//
//	// Login
//	err := client.Login(ctx, "user@example.com", "password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check permission
//	allowed, err := client.Can(ctx, userID, "edit", "invoice:123")
//
// # Token Persistence
//
// To persist tokens across restarts, use the WithOnTokenRefresh callback:
//
//	client := sdk.NewClient("https://auth.vyst.com.br",
//	    sdk.WithOnTokenRefresh(func(access, refresh string) {
//	        // Save tokens to secure storage
//	        saveTokens(access, refresh)
//	    }),
//	)
//
// To resume a session with saved tokens:
//
//	client := sdk.NewClient("https://auth.vyst.com.br",
//	    sdk.WithTokens(savedAccessToken, savedRefreshToken),
//	)
//
// # Permission Checks (ReBAC)
//
// The Can() method checks permissions using Vyst Identity's ReBAC engine:
//
//	// Can user edit this invoice?
//	allowed, _ := client.Can(ctx, userID, "edit", "invoice:123")
//
//	// Can user approve expenses over R$ 5000?
//	allowed, _ := client.Can(ctx, userID, "approve", "expense:high-value")
//
// # Thread Safety
//
// The Client is safe for concurrent use. Token refresh is handled
// automatically and atomically.
package sdk
