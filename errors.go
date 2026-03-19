package pihole

import "fmt"

type APIError struct {
	StatusCode int
	Key        string
	Message    string
	Hint       string
}

func (e *APIError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("pihole API error %d (%s): %s — %s", e.StatusCode, e.Key, e.Message, e.Hint)
	}
	return fmt.Sprintf("pihole API error %d (%s): %s", e.StatusCode, e.Key, e.Message)
}

type ErrNotFound struct {
	Resource string
	ID       string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

type ErrAuth struct {
	Message string
}

func (e *ErrAuth) Error() string {
	return fmt.Sprintf("authentication failed: %s", e.Message)
}
