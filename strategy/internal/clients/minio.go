package clients

import (
	"apex/internal/domain"
	"bytes"
	"context"
	"path"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const StrategyBucket = "strategies"

type Option func(*minio.Options)

func WithCredentials(username, password string) Option {
	return func(opts *minio.Options) {
		opts.Creds = credentials.NewStaticV4(username, password, "")
	}
}

func WithSecure(opts *minio.Options) {
	opts.Secure = true
}

type MinioClient struct {
	client *minio.Client
}

func New(endpoint string, opts ...Option) (*MinioClient, error) {
	options := minio.Options{
		Secure: false,
	}
	for _, opt := range opts {
		opt(&options)
	}

	client, err := minio.New(endpoint, &options)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, StrategyBucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, StrategyBucket, minio.MakeBucketOptions{ForceCreate: false}); err != nil {
			return nil, err
		}
	}

	return &MinioClient{client: client}, nil
}

func (mc *MinioClient) Upload(ctx context.Context, strategy *domain.Strategy, data []byte) (string, error) {
	reader := bytes.NewReader(data)
	path := path.Join(strategy.Name, strategy.FileName())

	info, err := mc.client.PutObject(ctx, StrategyBucket, path, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		return "", err
	}

	return info.Location, nil
}

func (mc *MinioClient) Download(ctx context.Context, path string) ([]byte, error) {
	obj, err := mc.client.GetObject(ctx, StrategyBucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(obj); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (mc *MinioClient) DownloadToDisk(ctx context.Context, strategy *domain.Strategy) (string, error) {
	path := path.Join(strategy.Name, strategy.FileName())
	if err := mc.client.FGetObject(ctx, StrategyBucket, strategy.FileName(), path, minio.GetObjectOptions{}); err != nil {
		return "", err
	}

	return path, nil
}
