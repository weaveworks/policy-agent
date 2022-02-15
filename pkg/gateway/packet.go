package gateway

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/MagalixTechnologies/uuid-go"
	"github.com/golang/snappy"
)

const (
	AuditResultStatusViolating = "Violation"
	AuditResultStatusCompliant = "Compliance"
	ProtocolMajorVersion       = 2
	ProtocolMinorVersion       = 4
)

type PacketKind string

const (
	PacketKindHello                PacketKind = "hello"
	PacketKindAuthorizationRequest PacketKind = "authorization/request"
	PacketKindAuthorizationAnswer  PacketKind = "authorization/answer"
	PacketKindAuditResult          PacketKind = "audit/result"
	PacketKindPing                 PacketKind = "ping"
)

func (kind PacketKind) String() string {
	return string(kind)
}

type PacketHello struct {
	Major     uint      `json:"major"`
	Minor     uint      `json:"minor"`
	Build     string    `json:"build"`
	AccountID uuid.UUID `json:"account_id"`
	ClusterID uuid.UUID `json:"cluster_id"`
}

type PacketAuthorizationRequest struct {
	AccountID uuid.UUID `json:"account_id"`
	ClusterID uuid.UUID `json:"cluster_id"`
}

type PacketAuthorizationQuestion struct {
	Token []byte `json:"token"`
}

type PacketAuthorizationAnswer struct {
	Token []byte `json:"token"`
}

type PacketAuthorizationFailure struct{}
type PacketAuthorizationSuccess struct{}

type PacketPing struct {
	Number  int       `json:"number,omitempty"`
	Started time.Time `json:"started"`
}

type PacketPong struct {
	Number  int       `json:"number,omitempty"`
	Started time.Time `json:"started"`
}

type AuditResultStatus string

type PacketAuditResultItem struct {
	ID            string                 `json:"id"`
	ConstraintID  string                 `json:"constraint_id"`
	CategoryID    string                 `json:"category_id"`
	Severity      string                 `json:"severity"`
	Controls      []string               `json:"controls"`
	Standards     []string               `json:"standards"`
	Description   string                 `json:"description"`
	HowToSolve    string                 `json:"how_to_solve"`
	Status        AuditResultStatus      `json:"status"`
	Msg           string                 `json:"msg"`
	EntityName    string                 `json:"entity_name"`
	EntityKind    string                 `json:"entity_kind"`
	NamespaceName string                 `json:"namespace_name,omitempty"`
	ParentName    string                 `json:"parent_name,omitempty"`
	ParentKind    string                 `json:"parent_kind,omitempty"`
	EntitySpec    map[string]interface{} `json:"entity_spec"`
	Trigger       string                 `json:"trigger"`
}

type PacketAuditResult struct {
	Items     []PacketAuditResultItem `json:"items"`
	Timestamp time.Time               `json:"timestamp"`
}

func EncodeSnappy(in interface{}) (out []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			err = fmt.Errorf("%s panic: %v", stack, r)
		}
	}()

	js, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("unable to encode to snappy, error: %w", err)
	}
	out = snappy.Encode(nil, js)
	return out, err
}

func DecodeSnappy(in []byte, out interface{}) error {
	js, err := snappy.Decode(nil, in)
	if err != nil {
		return fmt.Errorf("unable to decode to snappy, error: %w", err)
	}
	return json.Unmarshal(js, out)
}
