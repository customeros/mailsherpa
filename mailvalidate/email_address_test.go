package mailvalidate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsInvalidAddressError(t *testing.T) {
	tests := []struct {
		name        string
		description string
		errorCode   string
		want        bool
	}{
		{
			name:        "should detect invalid address by error code",
			description: "some generic error",
			errorCode:   "5.1.1",
			want:        true,
		},
		{
			name:        "should detect invalid address by description",
			description: "address does not exist",
			errorCode:   "",
			want:        true,
		},
		{
			name:        "should detect unknown user",
			description: "unknown user",
			errorCode:   "",
			want:        true,
		},
		{
			name:        "should not flag valid address error",
			description: "temporary server error",
			errorCode:   "4.0.0",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInvalidAddressError(tt.description, tt.errorCode)
			assert.Equal(t, tt.want, got, "isInvalidAddressError() returned unexpected result")
		})
	}
}

func TestIsMailboxFullError(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        bool
	}{
		{
			name:        "should detect insufficient storage",
			description: "Insufficient system storage",
			want:        true,
		},
		{
			name:        "should detect out of storage",
			description: "out of storage",
			want:        true,
		},
		{
			name:        "should detect over quota",
			description: "user is over quota",
			want:        true,
		},
		{
			name:        "should not flag non-storage error",
			description: "temporary server error",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMailboxFullError(tt.description)
			assert.Equal(t, tt.want, got, "isMailboxFullError() returned unexpected result")
		})
	}
}

func TestIsGreylistError(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        bool
	}{
		{
			name:        "should detect explicit greylisting",
			description: "greylisted",
			want:        true,
		},
		{
			name:        "should detect retry later message",
			description: "please retry later",
			want:        true,
		},
		{
			name:        "should detect postgrey",
			description: "postgrey in action",
			want:        true,
		},
		{
			name:        "should not flag non-greylist error",
			description: "temporary server error",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGreylistError(tt.description)
			assert.Equal(t, tt.want, got, "isGreylistError() returned unexpected result")
		})
	}
}

func TestIsBlacklistError(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        bool
	}{
		{
			name:        "should detect RBL block",
			description: "blocked by RBL",
			want:        true,
		},
		{
			name:        "should detect spamhaus",
			description: "listed in spamhaus",
			want:        true,
		},
		{
			name:        "should detect reputation block",
			description: "blocked due to poor reputation",
			want:        true,
		},
		{
			name:        "should not flag non-blacklist error",
			description: "temporary server error",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBlacklistError(tt.description)
			assert.Equal(t, tt.want, got, "isBlacklistError() returned unexpected result")
		})
	}
}

func TestDetermineGreylistDelay(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        int
	}{
		{
			name:        "should handle 4 minutes",
			description: "try again in 4 minutes",
			want:        6,
		},
		{
			name:        "should handle 5 minutes",
			description: "retry after 5 minutes",
			want:        6,
		},
		{
			name:        "should handle 360 seconds",
			description: "please wait 360 seconds",
			want:        7,
		},
		{
			name:        "should handle 60 seconds",
			description: "retry after 60 seconds",
			want:        2,
		},
		{
			name:        "should handle 1 minute",
			description: "try again in 1 minute",
			want:        2,
		},
		{
			name:        "should use default for unknown delay",
			description: "please try again later",
			want:        75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineGreylistDelay(tt.description)
			assert.Equal(t, tt.want, got, "determineGreylistDelay() returned unexpected delay")
		})
	}
}

func TestGetRetryTimestamp(t *testing.T) {
	tests := []struct {
		name         string
		minutesDelay int
	}{
		{
			name:         "should calculate future timestamp for 5 minutes",
			minutesDelay: 5,
		},
		{
			name:         "should calculate future timestamp for 60 minutes",
			minutesDelay: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().Unix()
			got := getRetryTimestamp(tt.minutesDelay)

			// Verify timestamp is in the future
			assert.Greater(t, got, int(now), "getRetryTimestamp() should return future timestamp")

			// Verify the delay is approximately correct (within 1 second tolerance)
			expectedDiff := tt.minutesDelay * 60
			actualDiff := got - int(now)
			assert.InDelta(t, expectedDiff, actualDiff, 1.0, "getRetryTimestamp() returned unexpected delay")
		})
	}
}

func TestHandleSmtpResponses(t *testing.T) {
	tests := []struct {
		name     string
		req      *EmailValidationRequest
		resp     *EmailValidation
		expected EmailValidation
	}{
		{
			name: "should handle deliverable response",
			req:  &EmailValidationRequest{},
			resp: &EmailValidation{
				IsDeliverable: "unknown",
				SmtpResponse: SmtpResponse{
					ResponseCode: "250",
				},
			},
			expected: EmailValidation{
				IsDeliverable:   "true",
				RetryValidation: false,
			},
		},
		{
			name: "should handle no MX record",
			req:  &EmailValidationRequest{},
			resp: &EmailValidation{
				IsDeliverable: "unknown",
				SmtpResponse: SmtpResponse{
					Description: "No MX records found",
				},
			},
			expected: EmailValidation{
				IsDeliverable:   "false",
				RetryValidation: false,
			},
		},
		{
			name: "should handle mailbox full",
			req:  &EmailValidationRequest{},
			resp: &EmailValidation{
				IsDeliverable: "unknown",
				SmtpResponse: SmtpResponse{
					ResponseCode: "452",
					Description:  "user is over quota",
				},
			},
			expected: EmailValidation{
				IsDeliverable:   "false",
				IsMailboxFull:   true,
				RetryValidation: false,
			},
		},
		{
			name: "should handle TLS required",
			req:  &EmailValidationRequest{},
			resp: &EmailValidation{
				IsDeliverable: "unknown",
				SmtpResponse: SmtpResponse{
					ResponseCode: "451",
					Description:  "TLS required for this connection",
				},
			},
			expected: EmailValidation{
				IsDeliverable:   "unknown",
				RetryValidation: true,
				SmtpResponse: SmtpResponse{
					TLSRequired: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleSmtpResponses(tt.req, tt.resp)

			assert.Equal(t, tt.expected.IsDeliverable, tt.resp.IsDeliverable, "IsDeliverable mismatch")
			assert.Equal(t, tt.expected.RetryValidation, tt.resp.RetryValidation, "RetryValidation mismatch")
			assert.Equal(t, tt.expected.IsMailboxFull, tt.resp.IsMailboxFull, "IsMailboxFull mismatch")
			assert.Equal(t, tt.expected.SmtpResponse.TLSRequired, tt.resp.SmtpResponse.TLSRequired, "TLSRequired mismatch")
		})
	}
}

func TestIsPermanentBlacklistError(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        bool
	}{
		{
			name:        "should detect access denied",
			description: "Access denied - invalid HELO name",
			want:        true,
		},
		{
			name:        "should detect bad reputation",
			description: "Bad reputation - listed in Spamhaus",
			want:        true,
		},
		{
			name:        "should detect barracuda",
			description: "rejected by barracudanetworks.com/reputation",
			want:        true,
		},
		{
			name:        "should detect RBL listing",
			description: "Your IP is listed in RBL",
			want:        true,
		},
		{
			name:        "should not flag temporary error",
			description: "Server temporarily unavailable",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPermanentBlacklistError(tt.description)
			assert.Equal(t, tt.want, got, "isPermanentBlacklistError() returned unexpected result")
		})
	}
}

func TestIsTemporaryBlockError(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        bool
	}{
		{
			name:        "should detect temporary block",
			description: "temporarily blocked",
			want:        true,
		},
		{
			name:        "should not flag permanent block",
			description: "permanently blocked",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTemporaryBlockError(tt.description)
			assert.Equal(t, tt.want, got, "isTemporaryBlockError() returned unexpected result")
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        bool
	}{
		{
			name:        "should detect try again message",
			description: "try again",
			want:        true,
		},
		{
			name:        "should not flag permanent error",
			description: "permanent failure",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableError(tt.description)
			assert.Equal(t, tt.want, got, "isRetryableError() returned unexpected result")
		})
	}
}

func TestIsDeliveryFailure(t *testing.T) {
	tests := []struct {
		name        string
		description string
		errorCode   string
		want        bool
	}{
		{
			name:        "should detect inbound disabled",
			description: "Account inbounds disabled",
			errorCode:   "",
			want:        true,
		},
		{
			name:        "should detect address rejected",
			description: "address rejected",
			errorCode:   "",
			want:        true,
		},
		{
			name:        "should detect by error code 4.4.4",
			description: "some error",
			errorCode:   "4.4.4",
			want:        true,
		},
		{
			name:        "should detect by error code 4.2.2",
			description: "some error",
			errorCode:   "4.2.2",
			want:        true,
		},
		{
			name:        "should not flag other errors",
			description: "temporary error",
			errorCode:   "4.0.0",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDeliveryFailure(tt.description, tt.errorCode)
			assert.Equal(t, tt.want, got, "isDeliveryFailure() returned unexpected result")
		})
	}
}
