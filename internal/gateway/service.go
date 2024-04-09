package gateway

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"github.com/spacelift-io/homework-object-storage/internal/discovery"
	"go.uber.org/zap"
)

// Service is the interface that provides the methods to interact with the S3 instances
type Service interface {
	AddOrUpdateObject(ctx context.Context, objectId string, data []byte) error
	GetObject(ctx context.Context, objectId string) ([]byte, error)
	Ready(ctx context.Context) bool
}

// ServiceV1 is the implementation of the Service interface
type ServiceV1 struct {
	discoveryService discovery.Service
	logger           *zap.Logger
}

// NewServiceV1 creates a new instance of the ServiceV1
func NewServiceV1(discoveryService discovery.Service) Service {
	return &ServiceV1{
		logger:           zap.L().Named("gateway"),
		discoveryService: discoveryService,
	}
}

// AddOrUpdateObject adds or updates an object in one of the available S3 instances
func (s *ServiceV1) AddOrUpdateObject(ctx context.Context, objectId string, data []byte) error {
	s.logger.Info("Adding or updating object in S3", zap.String("objectId", objectId))

	// Discover which S3 instances are available
	instances, err := s.discoveryService.DiscoverS3Instances(ctx)
	if err != nil {
		return err
	}

	// Determine which instance to read from based on the objectId

	// Minio client must be dynamically created, based on the S3 instance
	client, err := newMinioClient(instances[0])
	if err != nil {
		return err
	}

	// Put the object in the S3 instance
	_, err = client.PutObject(ctx, "mybucket", objectId, nil, int64(len(data)), minio.PutObjectOptions{})
	return err
}

// GetObject fetches an object from an instance of S3
func (s *ServiceV1) GetObject(ctx context.Context, objectId string) ([]byte, error) {
	s.logger.Info("Getting object from S3", zap.String("objectId", objectId))

	// Based on the ID, discover which S3 instance to use and fetch the object
	instances, err := s.discoveryService.DiscoverS3Instances(ctx)
	if err != nil {
		return nil, err
	}

	// Determine which instance to read from based on the objectId

	// Minio client must be dynamically created, based on the S3 instance
	client, err := newMinioClient(instances[0])
	if err != nil {
		return nil, err
	}

	// Get the object from the S3 instance
	_, err = client.GetObject(ctx, "mybucket", objectId, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Ready checks if the service is ready (if the Minio client is online and the Docker client is connected)
func (s *ServiceV1) Ready(ctx context.Context) bool {
	s.logger.Debug("Checking if the service is ready")
	if s.discoveryService == nil {
		return false
	}

	return s.discoveryService.Ready(ctx)
}

// newMinioClient creates a new instance of the Minio client based on the S3 instance
func newMinioClient(instance discovery.S3Instance) (*minio.Client, error) {
	minioClient, err := minio.New(instance.IpAddress, &minio.Options{
		Creds:  credentials.NewStaticV4("", instance.AccessKey, instance.SecretKey),
		Secure: false,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Minio client")
	}

	return minioClient, nil
}
