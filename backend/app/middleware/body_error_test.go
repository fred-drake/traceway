package middleware

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRejectBindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	addr := &net.TCPAddr{IP: net.ParseIP("10.1.2.3"), Port: 8082}
	cases := []struct {
		name        string
		err         error
		fallback    string
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "max bytes exceeded",
			err:         &http.MaxBytesError{Limit: 64},
			fallback:    "Invalid request body",
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantMessage: "Request body too large",
		},
		{
			name:        "body too slow",
			err:         ErrBodyTooSlow,
			fallback:    "Invalid request body",
			wantStatus:  http.StatusRequestTimeout,
			wantMessage: "Request body arrived too slowly",
		},
		{
			name:        "read deadline exceeded",
			err:         &net.OpError{Op: "read", Net: "tcp", Source: addr, Addr: addr, Err: os.ErrDeadlineExceeded},
			fallback:    "Invalid request body",
			wantStatus:  http.StatusRequestTimeout,
			wantMessage: "Request body arrived too slowly",
		},
		{
			name:        "plain bind error uses the fallback",
			err:         errors.New("bad json"),
			fallback:    "Invalid request body",
			wantStatus:  http.StatusBadRequest,
			wantMessage: "Invalid request body",
		},
		{
			name:        "plain bind error echoed through the fallback",
			err:         errors.New("bad json"),
			fallback:    "bad json",
			wantStatus:  http.StatusBadRequest,
			wantMessage: "bad json",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

			RejectBindError(c, tc.err, tc.fallback)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			var body struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body %q is not JSON: %v", rec.Body.String(), err)
			}
			if body.Error != tc.wantMessage {
				t.Fatalf("error = %q, want %q", body.Error, tc.wantMessage)
			}
			if strings.Contains(rec.Body.String(), addr.String()) {
				t.Fatalf("body %q leaks the listener address", rec.Body.String())
			}
		})
	}
}

func TestRejectIngestBindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name           string
		err            error
		wantStatus     int
		wantRetryAfter string
	}{
		{"body too slow becomes retryable", ErrBodyTooSlow, http.StatusServiceUnavailable, "2"},
		{"read deadline becomes retryable", ErrBodyTimedOut, http.StatusServiceUnavailable, "2"},
		{"max bytes stays 413", &http.MaxBytesError{Limit: 64}, http.StatusRequestEntityTooLarge, ""},
		{"plain bind error stays 400", errors.New("bad json"), http.StatusBadRequest, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

			RejectIngestBindError(c, tc.err, "Invalid request body")

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := rec.Header().Get("Retry-After"); got != tc.wantRetryAfter {
				t.Fatalf("Retry-After = %q, want %q", got, tc.wantRetryAfter)
			}
		})
	}
}
