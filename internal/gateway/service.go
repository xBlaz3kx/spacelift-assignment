package gateway

import (
	"context"
	docker "github.com/docker/docker/client"
	"github.com/minio/minio-go/v7"
)

type Service interface {
	AddOrUpdateObject(ctx context.Context, objectId string, data []byte) error
	GetObject(ctx context.Context, objectId string) ([]byte, error)
	DiscoverS3Instances(ctx context.Context) ([]string, error)
	Ready(ctx context.Context) bool
}

type ServiceV1 struct {
	minioClient  *minio.Client
	dockerClient *docker.Client
}

// AddOrUpdateObject adds or updates an object in one of the available S3 instances
func (s *ServiceV1) AddOrUpdateObject(ctx context.Context, objectId string, data []byte) error {
	// Discover which S3 instances are available and which one to use

	//TODO implement me
	panic("implement me")
}

// GetObject fetches an object from a instance of S3
func (s *ServiceV1) GetObject(ctx context.Context, objectId string) ([]byte, error) {
	// Based on the ID, discover which S3 instance to use and fetch the object
	//TODO implement me
	panic("implement me")
}

func (s *ServiceV1) DiscoverS3Instances(ctx context.Context) ([]string, error) {
	cli, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		panic(err)
	}

	//TODO implement me
	panic("implement me")
}

func (s *ServiceV1) Ready(ctx context.Context) bool {
	if s.minioClient == nil || s.dockerClient == nil {
		return false
	}

	// Try to ping docker
	_, err := s.dockerClient.Ping(ctx)

	// Check if Minio is online and if the ping was successful
	return s.minioClient.IsOnline() && err == nil
}

func NewServiceV1(minioClient *minio.Client, dockerClient *docker.Client) Service {
	return &ServiceV1{
		minioClient:  minioClient,
		dockerClient: dockerClient,
	}
}
