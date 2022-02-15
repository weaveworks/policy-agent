package gateway

import "fmt"

type FatalError struct {
	Err error
}

func (fe FatalError) Error() string {
	return fmt.Sprintf("fatal error, %s", fe.Err.Error())
}
