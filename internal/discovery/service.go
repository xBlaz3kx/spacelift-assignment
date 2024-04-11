package discovery

import (
	"context"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	s3ContainerPrefix = "amazin-object-storage-node-"
	minioPort         = "9000"
	minioAccessKey    = "MINIO_ACCESS_KEY="
	minioSecret       = "MINIO_SECRET_KEY="
)

type Service interface {
	DiscoverS3Instances(ctx context.Context) ([]S3Instance, error)
	Ready(ctx context.Context) bool
}

type ServiceV1 struct {
	dockerClient *docker.Client
	logger       *zap.Logger
}

func NewServiceV1(dockerClient *docker.Client) *ServiceV1 {
	return &ServiceV1{
		logger:       zap.L().Named("discovery"),
		dockerClient: dockerClient,
	}
}

// DiscoverS3Instances returns a list of available S3 instances from the Docker daemon, filtered by the prefix.
// Possible improvement - implement a cache for the instances, so we don't have to query Docker every time.
func (s *ServiceV1) DiscoverS3Instances(ctx context.Context) ([]S3Instance, error) {
	s.logger.Info("Discovering S3 instances")

	// Get the list of active containers - we will filter out the ones that are not S3 instances
	containers, err := s.dockerClient.ContainerList(ctx, container.ListOptions{All: false})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list containers")
	}

	response := []S3Instance{}

	for _, c := range containers {
		for _, name := range c.Names {
			// Include only the containers that have the "amazin-object-storage-node-" in their name
			if strings.Contains(name, s3ContainerPrefix) {
				s.logger.Info("Found an S3 instance container", zap.String("containerId", c.ID), zap.String("name", name))

				// Get the container details
				details, err := s.getContainerDetails(ctx, c.ID)
				if err != nil {
					return nil, err
				}

				response = append(response, *details)
				s.logger.Debug("Extracted container configuration", zap.Any("details", *details))
			}
		}
	}

	return response, nil
}

// getContainerDetails returns the details of a container
func (s *ServiceV1) getContainerDetails(ctx context.Context, containerId string) (*S3Instance, error) {
	s.logger.Info("Inspecting container", zap.String("containerId", containerId))

	inspectedContainer, err := s.dockerClient.ContainerInspect(ctx, containerId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to inspect container")
	}

	// Example: "/deployment-amazin-object-storage-node-2-1"
	// We need to trim any /<> from the name and remove the trailing -1 if it exists
	containerName := strings.Trim(inspectedContainer.Name, "/")
	nameParts := strings.Split(containerName, "-")
	prefix := s3ContainerPrefix
	if nameParts[0] != "amazin" {
		prefix = nameParts[0] + "-" + s3ContainerPrefix
	}

	// Extract the instance ID from the container name - the number after the prefix
	instanceIdString := strings.TrimSuffix(strings.TrimPrefix(containerName, prefix), "-1")
	instanceId, err := strconv.Atoi(instanceIdString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse instance ID")
	}

	// Extract the access key and secret key from the container environment
	s3AccessKey, s3SecretKey := "", ""
	for _, environmentVariable := range inspectedContainer.Config.Env {
		if strings.HasPrefix(environmentVariable, minioAccessKey) {
			s3AccessKey = strings.Split(environmentVariable, "=")[1]
		} else if strings.HasPrefix(environmentVariable, minioSecret) {
			s3SecretKey = strings.Split(environmentVariable, "=")[1]
		}
	}

	return &S3Instance{
		ContainerId: containerId,
		InstanceNum: instanceId,
		IpAddress:   inspectedContainer.NetworkSettings.IPAddress,
		Hostname:    inspectedContainer.Config.Hostname,
		AccessKey:   s3AccessKey,
		SecretKey:   s3SecretKey,
		// We can assume that the port is always 9000, since the upload/download will occur in the same docker network
		Port: minioPort,
	}, nil
}

// Ready checks if the service is ready (if Docker client is connected)
func (s *ServiceV1) Ready(ctx context.Context) bool {
	s.logger.Debug("Checking if the service is ready")
	if s.dockerClient == nil {
		return false
	}

	// Try to ping docker
	_, err := s.dockerClient.Ping(ctx)
	return err == nil
}
