package healthcheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// StatusEnum is a enum type for the valid states of a service.
type StatusEnum int

const (
	// OK is the status of a service when it is operating correctly.
	OK StatusEnum = iota
	// Warning is the status of a service when it is still available but some
	// systems may be unavailable or in an incorrect state.
	Warning
	// Critical is the status of a service in a critical state.
	Critical
)

// statusValues is a mapping of StatusEnum values to their human-readable form.
var statusValues = map[StatusEnum]string{
	OK:       "ok",
	Warning:  "warning",
	Critical: "critical",
}

// String converts a numeric status identifer to a human-readable
// representation, or returns `unknown` if the status can't be identified.
func (s StatusEnum) String() string {
	if status, ok := statusValues[s]; ok {
		return status
	}
	return "unknown"
}

// MarshalJSON is a concrete implementation of the json.Marshaler interface
// which allows StatusEnum values to be converted to their human-readable
// representation when serialising to JSON.
func (s StatusEnum) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// Checker is the interface to which all individual status checks must satisfy.
type Checker interface {
	Name() string
	Check() (StatusEnum, error)
}

// Status represents the overall status of the application.
type Status struct {
	Status StatusEnum       `json:"status"`
	Checks map[string]Check `json:"checks,omitEmpty"`
}

// Check is a single check which is performed as part of the overall system
// status.  All individual check statuses must be `ok` for the overall system
// status to be `ok`.
type Check struct {
	Status  StatusEnum `json:"status"`
	Message string     `json:"message,omitempty"`
}

// HealthCheck encapsulates and performs checks which are used to identify the
// health of a the application.
type HealthCheck struct {
	Timeout  time.Duration
	Checkers []Checker
}

// DefaultCheckTimeout is the default time period after which checks will be
// deemed to have failed.
const DefaultCheckTimeout = time.Second

// NewHealthCheck is a helper function for quickly creating a new HealthCheck
// value with the appropriate checkers in place and a default timeout.
func NewHealthCheck(checkers ...Checker) *HealthCheck {
	return &HealthCheck{Checkers: checkers, Timeout: DefaultCheckTimeout}
}

// Status runs all checks and responds with the individual statuses for those
// checks, as well as an overall status.
//
// * If all checks are `ok`, then the overall status will also be `ok`.
// * If one or more checks are in a `warning` state, and no checks are in a
//   `critical` state, then the overall status will be `warning`.
// * If one or more checks are in a `critical` state, the overall state will be
//   `critical`.
//
// If any check fails to return within a `HealthCheck.Timeout` duration then
// the check will be deemed to have failed.  In this situation, the individual
// check status will be set to `critical` and an appropraite message will be
// added.
func (h *HealthCheck) Status() Status {
	checked := map[string]Check{}
	status := OK
	chk := Check{}

	for _, checker := range h.Checkers {
		result := make(chan Check)

		go func() {
			c := Check{}

			var err error
			c.Status, err = checker.Check()

			if err != nil {
				c.Message = err.Error()
			}

			result <- c
		}()

		select {
		case c := <-result:
			chk = c
		case <-time.After(h.Timeout):
			chk = Check{
				Status:  Critical,
				Message: "Check timed out",
			}
		}

		if status < chk.Status {
			status = chk.Status
		}

		checked[checker.Name()] = chk
	}

	return Status{
		Status: status,
		Checks: checked,
	}
}

// HTTPHandler is a handler function for serving up the application healthcheck
// status via HTTP.
func (h *HealthCheck) HTTPHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		status := h.Status()
		encoder := json.NewEncoder(w)

		err := encoder.Encode(status)
		if err != nil {
			http.Error(w, fmt.Sprintf("Cannot encode response data: %v", err),
				http.StatusInternalServerError)
		}
	}
}
