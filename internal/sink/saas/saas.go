package saas

import (
	"context"
	"time"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/core/packet"
	"github.com/MagalixTechnologies/policy-core/domain"
)

// PolicyValidationSender sends data to the Saas gateway
type PolicyValidationSender interface {
	Send(kind packet.PacketKind, message interface{}, response interface{}) error
}

const (
	BufferSize          = 50
	sendRetriesInterval = 500 * time.Millisecond
	sendRetries         = 5
)

// SaaSGatewaySink batches policy validation and sends them to the Saas gateway
type SaaSGatewaySink struct {
	policyValidationSender  PolicyValidationSender
	packetKind              packet.PacketKind
	policyValidationsBatch  []domain.PolicyValidation
	policyValidationsBuffer chan domain.PolicyValidation
	saaSSinkBatchExpiry     time.Duration
}

// NewSaaSGatewaySink returns an instance of SaaSGatewaySink
func NewSaaSGatewaySink(
	policyValidationSender PolicyValidationSender,
	packetKind packet.PacketKind,
	saaSSinkBatchSize int,
	saaSSinkBatchExpiry time.Duration) *SaaSGatewaySink {
	batch := make([]domain.PolicyValidation, 0, saaSSinkBatchSize)
	buffer := make(chan domain.PolicyValidation, BufferSize)

	return &SaaSGatewaySink{
		policyValidationSender:  policyValidationSender,
		packetKind:              packetKind,
		policyValidationsBatch:  batch,
		policyValidationsBuffer: buffer,
		saaSSinkBatchExpiry:     saaSSinkBatchExpiry,
	}
}

// Write adds results to buffer, implements github.com/MagalixTechnologies/policy-core/domain.PolicyValidationSink
func (p *SaaSGatewaySink) Write(_ context.Context, policyValidations []domain.PolicyValidation) error {
	for i := range policyValidations {
		PolicyValidation := policyValidations[i]
		p.policyValidationsBuffer <- PolicyValidation
	}
	return nil
}

// Start starts the sink to send events when batch size is met or an interval has passed
func (p *SaaSGatewaySink) Start(ctx context.Context) error {
	timer := time.NewTicker(p.saaSSinkBatchExpiry)
	for {
		select {
		case result := <-p.policyValidationsBuffer:
			p.policyValidationsBatch = append(p.policyValidationsBatch, result)
			if len(p.policyValidationsBatch) == cap(p.policyValidationsBatch) {
				p.sendBatch(p.policyValidationsBatch)
				p.policyValidationsBatch = p.policyValidationsBatch[:0]
				timer.Reset(p.saaSSinkBatchExpiry)
			}
		case <-timer.C:
			if len(p.policyValidationsBatch) > 0 {
				p.sendBatch(p.policyValidationsBatch)
				p.policyValidationsBatch = p.policyValidationsBatch[:0]
			}
		case <-ctx.Done():
			if len(p.policyValidationsBatch) > 0 {
				p.sendBatch(p.policyValidationsBatch)
			}
			return ctx.Err()
		}
	}
}

func (p *SaaSGatewaySink) sendBatch(items []domain.PolicyValidation) {
	var err error
	logger.Infow("sending policy validations", "size", len(items))
	packet := packet.PacketPolicyValidationResults{
		Items:     items,
		Timestamp: time.Now().UTC(),
	}
	for i := 0; i < sendRetries; i++ {
		err = p.policyValidationSender.Send(p.packetKind, packet, nil)
		if err == nil {
			return
		}
		logger.Warnw("failed to send packet", "kind", p.packetKind, "retry", i+1, "error", err)
	}
	time.Sleep(sendRetriesInterval)
	logger.Errorf("failed to send packet of kind %s, lost %d items: %s", p.packetKind, len(packet.Items), err)
}
