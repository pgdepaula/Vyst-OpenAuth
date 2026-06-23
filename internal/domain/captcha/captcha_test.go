package captcha

import (
	"testing"
	"time"
)

func TestCaptchaType_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		captchaType CaptchaType
		want        bool
	}{
		{
			name:        "invisible type is valid",
			captchaType: TypeInvisible,
			want:        true,
		},
		{
			name:        "interactive type is valid",
			captchaType: TypeInteractive,
			want:        true,
		},
		{
			name:        "managed type is valid",
			captchaType: TypeManaged,
			want:        true,
		},
		{
			name:        "empty type is invalid",
			captchaType: CaptchaType(""),
			want:        false,
		},
		{
			name:        "unknown type is invalid",
			captchaType: CaptchaType("unknown"),
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.captchaType.IsValid(); got != tt.want {
				t.Errorf("CaptchaType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCaptchaType_String(t *testing.T) {
	tests := []struct {
		name        string
		captchaType CaptchaType
		want        string
	}{
		{
			name:        "invisible type string",
			captchaType: TypeInvisible,
			want:        "invisible",
		},
		{
			name:        "interactive type string",
			captchaType: TypeInteractive,
			want:        "interactive",
		},
		{
			name:        "managed type string",
			captchaType: TypeManaged,
			want:        "managed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.captchaType.String(); got != tt.want {
				t.Errorf("CaptchaType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewChallenge(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		siteKey     string
		captchaType CaptchaType
		ttl         time.Duration
		wantErr     bool
	}{
		{
			name:        "valid challenge",
			id:          "test-id-123",
			siteKey:     "test-site-key",
			captchaType: TypeManaged,
			ttl:         5 * time.Minute,
			wantErr:     false,
		},
		{
			name:        "missing id",
			id:          "",
			siteKey:     "test-site-key",
			captchaType: TypeManaged,
			ttl:         5 * time.Minute,
			wantErr:     true,
		},
		{
			name:        "missing site key",
			id:          "test-id",
			siteKey:     "",
			captchaType: TypeManaged,
			ttl:         5 * time.Minute,
			wantErr:     true,
		},
		{
			name:        "invalid type defaults to managed",
			id:          "test-id",
			siteKey:     "test-site-key",
			captchaType: CaptchaType("invalid"),
			ttl:         5 * time.Minute,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := NewChallenge(tt.id, tt.siteKey, tt.captchaType, tt.ttl)

			if tt.wantErr {
				if err == nil {
					t.Error("NewChallenge() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewChallenge() unexpected error: %v", err)
				return
			}

			if challenge.ID != tt.id {
				t.Errorf("Challenge.ID = %v, want %v", challenge.ID, tt.id)
			}
			if challenge.SiteKey != tt.siteKey {
				t.Errorf("Challenge.SiteKey = %v, want %v", challenge.SiteKey, tt.siteKey)
			}
			if challenge.CreatedAt.IsZero() {
				t.Error("Challenge.CreatedAt should not be zero")
			}
			if challenge.ExpiresAt.Before(challenge.CreatedAt) {
				t.Error("Challenge.ExpiresAt should be after CreatedAt")
			}
		})
	}
}

func TestChallenge_IsExpired(t *testing.T) {
	tests := []struct {
		name  string
		ttl   time.Duration
		sleep time.Duration
		want  bool
	}{
		{
			name:  "not expired - long TTL",
			ttl:   1 * time.Hour,
			sleep: 0,
			want:  false,
		},
		{
			name:  "expired - negative TTL",
			ttl:   -1 * time.Minute,
			sleep: 0,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := NewChallenge("test-id", "site-key", TypeManaged, tt.ttl)
			if err != nil {
				t.Fatalf("NewChallenge() error: %v", err)
			}

			if tt.sleep > 0 {
				time.Sleep(tt.sleep)
			}

			if got := challenge.IsExpired(); got != tt.want {
				t.Errorf("Challenge.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_IsHuman(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{
			name:   "high score human",
			result: Result{Success: true, Score: 0.9},
			want:   true,
		},
		{
			name:   "threshold score human",
			result: Result{Success: true, Score: 0.5},
			want:   true,
		},
		{
			name:   "low score bot",
			result: Result{Success: true, Score: 0.3},
			want:   false,
		},
		{
			name:   "failed validation with high score",
			result: Result{Success: false, Score: 0.9},
			want:   false,
		},
		{
			name:   "zero score",
			result: Result{Success: true, Score: 0.0},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsHuman(); got != tt.want {
				t.Errorf("Result.IsHuman() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_IsBot(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{
			name:   "high score is not bot",
			result: Result{Success: true, Score: 0.9},
			want:   false,
		},
		{
			name:   "low score is bot",
			result: Result{Success: true, Score: 0.3},
			want:   true,
		},
		{
			name:   "failed validation is bot",
			result: Result{Success: false, Score: 0.9},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsBot(); got != tt.want {
				t.Errorf("Result.IsBot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_HasErrors(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{
			name:   "no errors",
			result: Result{ErrorCodes: nil},
			want:   false,
		},
		{
			name:   "empty error codes",
			result: Result{ErrorCodes: []string{}},
			want:   false,
		},
		{
			name:   "has error codes",
			result: Result{ErrorCodes: []string{"missing-input-response"}},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasErrors(); got != tt.want {
				t.Errorf("Result.HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewResult(t *testing.T) {
	result := NewResult(true, 0.8)

	if !result.Success {
		t.Error("NewResult() Success should be true")
	}
	if result.Score != 0.8 {
		t.Errorf("NewResult() Score = %v, want %v", result.Score, 0.8)
	}
	if result.ChallengeTS.IsZero() {
		t.Error("NewResult() ChallengeTS should not be zero")
	}
}

// Test that domain errors are properly defined
func TestDomainErrors(t *testing.T) {
	errors := []struct {
		name string
		err  error
	}{
		{"ErrCaptchaRequired", ErrCaptchaRequired},
		{"ErrCaptchaExpired", ErrCaptchaExpired},
		{"ErrCaptchaInvalid", ErrCaptchaInvalid},
		{"ErrCaptchaTokenMissing", ErrCaptchaTokenMissing},
	}

	for _, e := range errors {
		t.Run(e.name, func(t *testing.T) {
			if e.err == nil {
				t.Errorf("%s should not be nil", e.name)
			}
			if e.err.Error() == "" {
				t.Errorf("%s should have an error message", e.name)
			}
		})
	}
}
