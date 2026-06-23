package main

import (
	"log"
	"net/http"
	"time"

	"github.com/pgdepaula/vyst-openauth/pkg/sdk"
)

var (
	vystURL = "http://localhost:8080"
)

func main() {
	// Initialize SDK Client
	client := sdk.NewClient(vystURL)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
		<html>
			<body>
				<h1>Vyst Identity Demo App</h1>
				<p>This is a demo application to test Vyst Identity SDK.</p>
				<form action="/login" method="post">
					<input type="email" name="email" placeholder="Email" required>
					<input type="password" name="password" placeholder="Password" required>
					<button type="submit">Login</button>
				</form>
			</body>
		</html>
		`
		_, _ = w.Write([]byte(html))
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		// Perform login using SDK
		// Note: In a real OIDC flow, we would redirect the user to Vyst Identity.
		// Since the SDK currently supports direct login (Resource Owner Password Credentials or similar),
		// we will use that for this demo as per the current SDK implementation.
		err := client.Login(r.Context(), email, password)
		if err != nil {
			http.Error(w, "Login failed: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Set cookie or session (simplified for demo)
		http.SetCookie(w, &http.Cookie{
			Name:  "access_token",
			Value: client.GetAccessToken(),
			Path:  "/",
		})

		http.Redirect(w, r, "/protected", http.StatusSeeOther)
	})

	http.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("access_token")
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Validate token (using SDK introspection or local validation if keys were available)
		// Here we just use the SDK to make a call to /auth/me to verify the token works

		// Create a new client with the token for this request
		reqClient := sdk.NewClient(vystURL, sdk.WithTokens(cookie.Value, ""))

		resp, err := reqClient.Get(r.Context(), "/auth/me")
		if err != nil {
			http.Error(w, "Failed to call API: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		_, _ = w.Write([]byte("<h1>Protected Area</h1><p>You are logged in!</p><a href='/'>Logout</a>"))
	})

	log.Println("Demo app starting on :3001")
	server := &http.Server{
		Addr:              ":3001",
		Handler:           nil,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
