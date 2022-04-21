package gateway

import "fmt"

// ConnectionError represents a gateway connection error
type ConnectionError struct {
	Err error
}

func (fe ConnectionError) Error() string {
	return fmt.Sprintf("fatal error, %s", fe.Err)
}
