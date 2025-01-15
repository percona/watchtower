package update

import (
	"errors"
	"github.com/containrrr/watchtower/pkg/types"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleError(t *testing.T) {
	tests := []struct {
		name           string
		error          error
		expectedStatus int
	}{
		{
			name:           "no error returns OK",
			error:          nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "validation error returns message",
			error:          types.NewValidationError("no new image available"),
			expectedStatus: http.StatusPreconditionFailed,
		},
		{
			name:           "generic server error",
			error:          errors.New("server error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = httptest.NewRequest(http.MethodGet, "/v1/update", nil)
			rr := httptest.NewRecorder()

			handleError(rr, tt.error)

			if status := rr.Result().StatusCode; status != tt.expectedStatus {
				t.Errorf("the handler wrote status code %d, expected: %d", status, tt.expectedStatus)
			}
		})
	}
}
