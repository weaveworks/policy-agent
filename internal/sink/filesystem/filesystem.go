package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/policy-core/domain"
)

const (
	kubernetespProvider = "Kubernetes"
)

type FileSystemSink struct {
	File                 *os.File
	AccountID            string
	ClusterID            string
	PolicyValidationChan chan domain.PolicyValidation
	cancelWorker         context.CancelFunc
}

// NewFileSystemSink returns a sink that writes results to the file system
func NewFileSystemSink(filePath string, accountID, clusterID string) (*FileSystemSink, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s to write validation results: %w", filePath, err)
	}
	return &FileSystemSink{
		File:                 file,
		AccountID:            accountID,
		ClusterID:            clusterID,
		PolicyValidationChan: make(chan domain.PolicyValidation, 50),
	}, nil
}

// Start starts the writer worker
func (f *FileSystemSink) Start(ctx context.Context) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	f.cancelWorker = cancel
	return f.WritePolicyValidationWorker(cancelCtx)
}

func (f *FileSystemSink) writeValidationResutl(policyValidation domain.PolicyValidation) error {
	err := json.NewEncoder(f.File).Encode(policyValidation)
	if err != nil {
		return fmt.Errorf("failed to write result to file: %w", err)
	}
	return nil
}

// WritePolicyValidationWorker worker that listens on results and admits them to a file
func (f *FileSystemSink) WritePolicyValidationWorker(_ context.Context) error {
	for {
		select {
		case result := <-f.PolicyValidationChan:
			err := f.writeValidationResutl(result)
			if err != nil {
				logger.Errorw(
					"error while writing results",
					"error", err,
					"policy-id", result.Policy.ID,
					"entity-name", result.Entity.Name,
					"entity-type", result.Entity.Kind,
					"status", result.Status,
				)
			}
		}
	}
}

// Write adds results to buffer, implements github.com/MagalixTechnologies/policy-core/domain.PolicyValidationSink
func (f *FileSystemSink) Write(_ context.Context, policyValidations []domain.PolicyValidation) error {
	for i := range policyValidations {
		PolicyValidation := policyValidations[i]
		f.PolicyValidationChan <- PolicyValidation
	}

	return nil
}

// Stop stops file writer worker and commits all results to disk
func (f *FileSystemSink) Stop() error {
	defer f.File.Close()

	f.cancelWorker()
	err := f.File.Sync()
	if err != nil {
		msg := fmt.Sprintf("failed to write all validations results to file, %s", err)
		logger.Error(msg)
		return fmt.Errorf(msg)
	}
	return nil
}
