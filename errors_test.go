package sdk

import (
	"errors"
	"testing"
)

type hostCauseError struct {
	message string
}

func (e *hostCauseError) Error() string { return e.message }

func TestHostStatusError(t *testing.T) {
	t.Parallel()

	cause := &hostCauseError{message: "permission denied"}
	hostCallErr := errors.New("transport failed")
	tests := []struct {
		name       string
		err        *HostStatusError
		wantString string
		wantHost   bool
		wantCause  bool
	}{
		{
			name:       "default target",
			err:        &HostStatusError{},
			wantString: "host operation: host returned an error status",
		},
		{
			name:       "capability only",
			err:        &HostStatusError{Capability: "kvstore"},
			wantString: "kvstore: host returned an error status",
		},
		{
			name:       "operation only",
			err:        &HostStatusError{Operation: "get"},
			wantString: "get: host returned an error status",
		},
		{
			name:       "capability operation and cause",
			err:        &HostStatusError{Capability: "kvstore", Operation: "get", Cause: cause},
			wantString: "kvstore/get: host returned an error status: permission denied",
			wantCause:  true,
		},
		{
			name:       "hostcall chain",
			err:        &HostStatusError{Capability: "sql", Operation: "query", HostCallErr: hostCallErr},
			wantString: "sql/query: host returned an error status",
			wantHost:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.err.Error(); got != tc.wantString {
				t.Fatalf("Error() = %q, want %q", got, tc.wantString)
			}
			if !errors.Is(tc.err, ErrHostError) {
				t.Fatal("errors.Is(error, ErrHostError) = false, want true")
			}
			var statusErr *HostStatusError
			if !errors.As(tc.err, &statusErr) {
				t.Fatal("errors.As(error, *HostStatusError) = false, want true")
			}
			if got := errors.Is(tc.err, ErrHostCall); got != tc.wantHost {
				t.Fatalf("errors.Is(error, ErrHostCall) = %t, want %t", got, tc.wantHost)
			}
			if tc.wantHost && !errors.Is(tc.err, hostCallErr) {
				t.Fatal("errors.Is(error, hostCallErr) = false, want true")
			}
			if got := errors.Is(tc.err, cause); got != tc.wantCause {
				t.Fatalf("errors.Is(error, cause) = %t, want %t", got, tc.wantCause)
			}

			var typedCause *hostCauseError
			if got := errors.As(tc.err, &typedCause); got != tc.wantCause {
				t.Fatalf("errors.As(error, *hostCauseError) = %t, want %t", got, tc.wantCause)
			}
		})
	}
}
