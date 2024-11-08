package gateway

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"mime/multipart"
	"sync"

	"github.com/pkg/errors"
	"github.com/spacelift-io/homework-object-storage/internal/discovery"
	"github.com/spacelift-io/homework-object-storage/internal/pkg/s3"
	"go.uber.org/zap"
)

// Service is the interface that provides the methods to interact with the S3 instances
type Service interface {
	AddOrUpdateObject(ctx context.Context, objectId string, file multipart.File) error
	GetObject(ctx context.Context, objectId string) (io.Reader, error)
	GetObjects(ctx context.Context) ([]string, error)
	GetObjectsAsync(ctx context.Context) ([]string, error)
	Ready(ctx context.Context) bool
	shardObjectToInstance(ctx context.Context, objectId string) (*discovery.S3Instance, error)
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
	logger := s.logger.With(zap.String("objectId", objectId))
	logger.Info("Adding or updating object in S3")

	// Determine which instance to write to based on the objectId
	instance, err := s.shardObjectToInstance(ctx, objectId)
	if err != nil {
		return errors.Wrap(err, "failed to assign object to instance")
	}

	// Minio client must be dynamically created, based on the S3 instance
	client, err := s3.NewMinioClient(*instance)
	if err != nil {
		return err
	}

	logger.Info("Adding object to S3 instance", zap.Int("instance", instance.InstanceNum))
	return client.AddOrUpdateObject(ctx, objectId, data)
}

// GetObject fetches an object from an instance of S3
func (s *ServiceV1) GetObject(ctx context.Context, objectId string) (io.Reader, error) {
	logger := s.logger.With(zap.String("objectId", objectId))
	logger.Info("Getting object from S3")

	// Determine which instance to read from based on the objectId
	instance, err := s.shardObjectToInstance(ctx, objectId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to assign object to instance")
	}

	// Minio client must be dynamically created, based on the S3 instance
	client, err := s3.NewMinioClient(*instance)
	if err != nil {
		return nil, err
	}

	logger.Info("Getting object from S3 instance", zap.Int("instance", instance.InstanceNum))

	// Get the object from the S3 instance
	obj, err := client.GetObject(ctx, objectId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get object from S3")
	}

	return obj, nil
}

// GetObjects get all objects (from all instances)
func (s *ServiceV1) GetObjects(ctx context.Context) ([]string, error) {
	s.logger.Info("Get all objects")

	// Discover available S3 instances
	instances, err := s.discoveryService.DiscoverS3Instances(ctx)
	if err != nil {
		return nil, err
	}

	objectIds := []string{}

	for _, instance := range instances {
		// Minio client must be dynamically created, based on the S3 instance
		client, err := s3.NewMinioClient(instance)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to create s3 client for instance: %d", instance.InstanceNum))
		}

		objects, err := client.GetObjects(ctx)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to list objectIds for instance: %d", instance.InstanceNum))
		}

		objectIds = append(objectIds, objects...)
	}

	return objectIds, nil
}

// GetObjects get all objects from all instances asnychonously
func (s *ServiceV1) GetObjectsAsync(ctx context.Context) ([]string, error) {
	s.logger.Info("Get all objects")

	// Discover available S3 instances
	instances, err := s.discoveryService.DiscoverS3Instances(ctx)
	if err != nil {
		return nil, err
	}

	// ObjectIds need to be accessed in a thread-safe way
	objectIds := []string{}
	objectIdMutex := sync.Mutex{}

	// Create a wait group and an error channel
	var wg sync.WaitGroup
	errChan := make(chan error, len(instances))

	for _, instance := range instances {
		wg.Add(1)

		go func(s3Instance discovery.S3Instance) {
			defer wg.Done()

			// Minio client must be dynamically created, based on the S3 instance
			client, err := s3.NewMinioClient(s3Instance)
			if err != nil {
				errChan <- errors.Wrap(err, fmt.Sprintf("unable to create s3 client for instance: %d", s3Instance.InstanceNum))
				return
			}

			objects, err := client.GetObjects(ctx)
			if err != nil {
				errChan <- errors.Wrap(err, fmt.Sprintf("unable to list objectIds for instance: %d", s3Instance.InstanceNum))
				return
			}

			objectIdMutex.Lock()
			objectIds = append(objectIds, objects...)
			objectIdMutex.Unlock()
		}(instance)
	}

	// Wait for all workers to finish.
	wg.Wait()

	// Check if an error occurred in any of the workers.
	close(errChan)
	for readErr := range errChan {
		if !errors.Is(readErr, nil) {
			return nil, readErr
		}
	}

	return objectIds, nil
}

// Ready checks if the service is ready (if the Minio client is online and the Docker client is connected)
func (s *ServiceV1) Ready(ctx context.Context) bool {
	s.logger.Debug("Checking if the service is ready")
	if s.discoveryService == nil {
		return false
	}

	return s.discoveryService.Ready(ctx)
}

// shardObjectToInstance chooses an instance to write an object to. A form of sharding is used to determine the instance.
func (s *ServiceV1) shardObjectToInstance(ctx context.Context, objectId string) (*discovery.S3Instance, error) {
	s.logger.Debug("Assigning object to instance", zap.String("objectId", objectId))

	// Discover available S3 instances
	instances, err := s.discoveryService.DiscoverS3Instances(ctx)
	if err != nil {
		return nil, err
	}

	// If there are no instances available, return an error
	if len(instances) == 0 {
		return nil, errors.New("no instances available")
	}

	// Hash the objectId and use the modulo of the hash to determine the instance
	// https://medium.com/@nynptel/what-is-modular-hashing-9c1fbbb3c611
	objectIdHash := hashId(objectId)
	instanceNum := objectIdHash % uint64(len(instances))

	// Get the instance based on the instanceNum -> Fixed the function based on feedback
	for _, instance := range instances {
		if instance.InstanceNum == int(instanceNum) {
			return &instance, nil
		}
	}

	return nil, errors.New("instance not found or unavailable")
}

func hashId(id string) uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(id))
	return hash.Sum64()
}
