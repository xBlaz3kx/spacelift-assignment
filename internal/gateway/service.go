package gateway

import (
	"context"
	"hash/fnv"
	"io"
	"mime/multipart"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"github.com/spacelift-io/homework-object-storage/internal/discovery"
	"go.uber.org/zap"
)

const bucketName = "spacelift-storage"

// Service is the interface that provides the methods to interact with the S3 instances
type Service interface {
	AddOrUpdateObject(ctx context.Context, objectId string, file multipart.File) error
	GetObject(ctx context.Context, objectId string) (io.Reader, error)
	Ready(ctx context.Context) bool
	assignObjectToInstance(ctx context.Context, objectId string, instances []discovery.S3Instance) (*discovery.S3Instance, error)
}

// ServiceV1 is the implementation of the Service interface
type ServiceV1 struct {
	discoveryService discovery.Service
	logger           *zap.Logger
}

// NewServiceV1 creates a new instance of the ServiceV1
func NewServiceV1(discoveryService discovery.Service) *ServiceV1 {
	return &ServiceV1{
		logger:           zap.L().Named("gateway"),
		discoveryService: discoveryService,
	}
}

// AddOrUpdateObject adds or updates an object in one of the available S3 instances
func (s *ServiceV1) AddOrUpdateObject(ctx context.Context, objectId string, data multipart.File) error {
	s.logger.Info("Adding or updating object in S3", zap.String("objectId", objectId))

	// Discover which S3 instances are available
	instances, err := s.discoveryService.DiscoverS3Instances(ctx)
	if err != nil {
		return err
	}

	// Determine which instance to write to based on the objectId
	instance, err := s.assignObjectToInstance(ctx, objectId, instances)
	if err != nil {
		return errors.Wrap(err, "failed to assign object to instance")
	}

	// Minio client must be dynamically created, based on the S3 instance
	client, err := newMinioClient(*instance)
	if err != nil {
		return err
	}

	// Check if the bucket exists, if not create it
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return errors.Wrap(err, "failed to check if bucket exists")
	}
	if !exists {
		// Check if bucket exists, if not create it
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create a new bucket")
		}
	}

	// Put the object in the S3 instance
	_, err = client.PutObject(ctx, bucketName, objectId, data, 1, minio.PutObjectOptions{})
	return err
}

// GetObject fetches an object from an instance of S3
func (s *ServiceV1) GetObject(ctx context.Context, objectId string) (io.Reader, error) {
	s.logger.Info("Getting object from S3", zap.String("objectId", objectId))

	// Based on the ID, discover which S3 instance to use and fetch the object
	instances, err := s.discoveryService.DiscoverS3Instances(ctx)
	if err != nil {
		return nil, err
	}

	// Determine which instance to read from based on the objectId
	instance, err := s.assignObjectToInstance(ctx, objectId, instances)
	if err != nil {
		return nil, errors.Wrap(err, "failed to assign object to instance")
	}

	// Minio client must be dynamically created, based on the S3 instance
	client, err := newMinioClient(*instance)
	if err != nil {
		return nil, err
	}

	// Get the object from the S3 instance
	obj, err := client.GetObject(ctx, bucketName, objectId, minio.GetObjectOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get object from S3")
	}

	return obj, nil
}

// Ready checks if the service is ready (if the Minio client is online and the Docker client is connected)
func (s *ServiceV1) Ready(ctx context.Context) bool {
	s.logger.Debug("Checking if the service is ready")
	if s.discoveryService == nil {
		return false
	}

	return s.discoveryService.Ready(ctx)
}

// assignObjectToInstance chooses an instance to write an object to. A form of sharding is used to determine the instance.
func (s *ServiceV1) assignObjectToInstance(ctx context.Context, objectId string, instances []discovery.S3Instance) (*discovery.S3Instance, error) {
	s.logger.Debug("Assigning object to instance", zap.String("objectId", objectId))

	// Hash the objectId and use the modulo of the hash to determine the instance
	// https://medium.com/@nynptel/what-is-modular-hashing-9c1fbbb3c611
	objectIdHash := hashId(objectId)
	instanceNum := objectIdHash % uint64(len(instances))

	return &instances[instanceNum], nil
}

func hashId(id string) uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(id))
	return hash.Sum64()
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
