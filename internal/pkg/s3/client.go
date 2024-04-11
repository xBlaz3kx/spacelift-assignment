package s3

import (
	"context"
	"io"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"github.com/spacelift-io/homework-object-storage/internal/discovery"
	"go.uber.org/zap"
)

var ErrObjectNotFound = errors.New("object not found")

const bucketName = "spacelift-storage"

type Client interface {
	AddOrUpdateObject(ctx context.Context, objectId string, data io.Reader) error
	GetObject(ctx context.Context, objectId string) (io.Reader, error)
}

type MinioClient struct {
	client *minio.Client
	logger *zap.Logger
}

// NewMinioClient creates a new instance of the Minio client based on the S3 instance
func NewMinioClient(instance discovery.S3Instance) (*MinioClient, error) {
	minioClient, err := minio.New(instance.IpAddress, &minio.Options{
		Creds:  credentials.NewStaticV4("", instance.AccessKey, instance.SecretKey),
		Secure: false,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Minio client")
	}

	return &MinioClient{
		client: minioClient,
		logger: zap.L().Named("minio-client"),
	}, nil
}

func (c *MinioClient) AddOrUpdateObject(ctx context.Context, objectId string, data io.Reader) error {
	c.logger.Info("Adding or updating object in S3", zap.String("objectId", objectId))

	// Check if the bucket exists, if not create it
	exists, err := c.client.BucketExists(ctx, bucketName)
	if err != nil {
		return errors.Wrap(err, "failed to check if bucket exists")
	}
	if !exists {

		// Check if bucket exists, if not create it
		err = c.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create a new bucket")
		}
	}

	// Put the object in the S3 instance
	_, err = c.client.PutObject(ctx, bucketName, objectId, data, 1, minio.PutObjectOptions{})
	if err != nil {

		res := minio.ToErrorResponse(err)
		if res.StatusCode == http.StatusNotFound {
			return ErrObjectNotFound
		}
	}

	return nil
}

func (c *MinioClient) GetObject(ctx context.Context, objectId string) (io.Reader, error) {
	c.logger.Info("Adding or updating object in S3", zap.String("objectId", objectId))

	// Get the object from the S3 instance
	obj, err := c.client.GetObject(ctx, bucketName, objectId, minio.GetObjectOptions{})
	if err != nil {
		res := minio.ToErrorResponse(err)
		if res.StatusCode == http.StatusNotFound {
			return nil, ErrObjectNotFound
		}

		return nil, errors.Wrap(err, "failed to get object from S3")
	}

	return obj, nil
}
