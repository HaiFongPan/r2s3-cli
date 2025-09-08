package r2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	
	appconfig "github.com/HaiFongPan/r2s3-cli/internal/config"
)

// Client wraps the S3 client for R2 operations
type Client struct {
	s3Client *s3.Client
	config   *appconfig.R2Config
}

// NewClient creates a new R2 client from configuration
func NewClient(cfg *appconfig.R2Config) (*Client, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.AccessKeySecret,
			"",
		)),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID))
	})

	return &Client{
		s3Client: s3Client,
		config:   cfg,
	}, nil
}

// GetS3Client returns the underlying S3 client
func (c *Client) GetS3Client() *s3.Client {
	return c.s3Client
}

// GetBucketName returns the configured bucket name
func (c *Client) GetBucketName() string {
	return c.config.BucketName
}