package saas

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MagalixTechnologies/core/packet"
	"github.com/MagalixTechnologies/policy-core/domain"
)

type SenderMock struct {
	err       error
	numCalled int
}

func (s *SenderMock) Send(kind packet.PacketKind, message interface{}, response interface{}) error {
	s.numCalled++
	if s.err != nil {
		return s.err
	}
	return nil
}

func TestPolicyValidationSinkSend(t *testing.T) {
	type args struct {
		policyValidationSender       PolicyValidationSender
		packetKind                   packet.PacketKind
		policyValidationsBatchSize   int
		policyValidationsBatchExpiry time.Duration
	}
	tests := []struct {
		name          string
		args          args
		expectedCalls int
	}{
		{
			name: "reached batch size",
			args: args{
				policyValidationSender:       &SenderMock{},
				packetKind:                   packet.PacketPolicyValidationAudit,
				policyValidationsBatchSize:   1,
				policyValidationsBatchExpiry: 5 * time.Second,
			},
			expectedCalls: 1,
		},
		{
			name: "reached batch expiry",
			args: args{
				policyValidationSender:       &SenderMock{},
				packetKind:                   packet.PacketPolicyValidationAudit,
				policyValidationsBatchSize:   2,
				policyValidationsBatchExpiry: 1 * time.Second,
			},
			expectedCalls: 1,
		},
		{
			name: "retry send on error",
			args: args{
				policyValidationSender:       &SenderMock{err: errors.New("")},
				packetKind:                   packet.PacketPolicyValidationAudit,
				policyValidationsBatchSize:   1,
				policyValidationsBatchExpiry: 5 * time.Second,
			},
			expectedCalls: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSaaSGatewaySink(
				tt.args.policyValidationSender,
				tt.args.packetKind,
				tt.args.policyValidationsBatchSize,
				tt.args.policyValidationsBatchExpiry)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go s.Start(ctx)
			s.Write(ctx, []domain.PolicyValidation{{}})
			time.Sleep(2 * time.Second)
			numCalled := tt.args.policyValidationSender.(*SenderMock).numCalled
			if tt.expectedCalls != numCalled {
				t.Errorf(
					"got unexpected number of calls to policy validation send in test %s. got %d expected %d",
					tt.name,
					numCalled,
					tt.expectedCalls)
			}
		})
	}
}
