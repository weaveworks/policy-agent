package core

import (
	"encoding/json"
	"fmt"

	"github.com/open-policy-agent/opa/ast"
)

// Policy contains policy and metedata
type Policy struct {
	module *ast.Module
	pkg    string
}

type OPAError interface {
	GetDetails() interface{}
}

// NoValidError indicates
type NoValidError struct {
	Details interface{}
}

func (e NoValidError) Error() string {
	details, err := json.Marshal(e.Details)
	if err != nil {
		return fmt.Sprintf("error while parsing error details: %+v", err)
	}
	return string(details)
}

func (e NoValidError) GetDetails() interface{} {
	return e.Details
}
