package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/MagalixTechnologies/core/logger"
)

const (
	kubernetespProvider = "Kubernetes"
)

type FileSystemSink struct {
	File                 *os.File
	AccountID            string
	ClusterID            string
	validationResultChan chan domain.ValidationResult
	cancelWorker         context.CancelFunc
}

// NewFileSystemSink returns a sink that writes results to the file system
func NewFileSystemSink(filePath string, accountID, clusterID string) (*FileSystemSink, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s to write validation results, %w", filePath, err)
	}
	return &FileSystemSink{
		File:                 file,
		AccountID:            accountID,
		ClusterID:            clusterID,
		validationResultChan: make(chan domain.ValidationResult, 50),
	}, nil
}

// Start starts the writer worker
func (f *FileSystemSink) Start(ctx context.Context) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	f.cancelWorker = cancel
	go f.WriteValidationResultWorker(cancelCtx)
	return nil
}

func (f *FileSystemSink) writeValidationResutl(validationResult domain.ValidationResult) error {
	result := Result{
		ID:         validationResult.ID,
		AccountID:  f.AccountID,
		ClusterID:  f.ClusterID,
		PolicyID:   validationResult.Policy.ID,
		Status:     validationResult.Status,
		Type:       validationResult.Source,
		Provider:   kubernetespProvider,
		EntityName: validationResult.Entity.Name,
		EntityType: validationResult.Entity.Kind,
		CreatedAt:  validationResult.CreatedAt,
		Message:    validationResult.Message,
		Info:       map[string]interface{}{"spec": validationResult.Entity.Spec},
		CategoryID: validationResult.Policy.Category,
		Severity:   validationResult.Policy.Severity,
	}
	err := json.NewEncoder(f.File).Encode(result)
	if err != nil {
		return fmt.Errorf("faile to write result to file, %w", err)
	}
	return nil
}

// WriteValidationResultWorker worker that listens on results and admits them to a file
func (f *FileSystemSink) WriteValidationResultWorker(ctx context.Context) {
	for {
		select {
		case result := <-f.validationResultChan:
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

// Write adds results to buffer, implements github.com/MagalixCorp/magalix-policy-agent/pkg/domain.ValidationResultSink
func (f *FileSystemSink) Write(ctx context.Context, validationResults []domain.ValidationResult) error {
	for i := range validationResults {
		validationResult := validationResults[i]
		f.validationResultChan <- validationResult
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
