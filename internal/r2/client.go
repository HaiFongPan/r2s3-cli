package r2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

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

// ListBuckets lists all buckets in the R2 account
func (c *Client) ListBuckets(ctx context.Context) ([]types.Bucket, error) {
	input := &s3.ListBucketsInput{}
	
	result, err := c.s3Client.ListBuckets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	
	return result.Buckets, nil
}

// GetBucketLocation gets the region/location of a specific bucket
func (c *Client) GetBucketLocation(ctx context.Context, bucketName string) (string, error) {
	input := &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	}
	
	result, err := c.s3Client.GetBucketLocation(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket location for %s: %w", bucketName, err)
	}
	
	// Handle empty location constraint (default region)
	if result.LocationConstraint == "" {
		return "us-east-1", nil
	}
	
	return string(result.LocationConstraint), nil
}

// HeadBucket checks if a bucket exists and is accessible
func (c *Client) HeadBucket(ctx context.Context, bucketName string) error {
	input := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}
	
	_, err := c.s3Client.HeadBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to head bucket %s: %w", bucketName, err)
	}
	
	return nil
}

// GetBucketPolicy retrieves the bucket policy for a specific bucket
func (c *Client) GetBucketPolicy(ctx context.Context, bucketName string) (*s3.GetBucketPolicyOutput, error) {
	input := &s3.GetBucketPolicyInput{
		Bucket: aws.String(bucketName),
	}
	
	result, err := c.s3Client.GetBucketPolicy(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket policy for %s: %w", bucketName, err)
	}
	
	return result, nil
}

// GetBucketWebsite retrieves the website configuration for a specific bucket
func (c *Client) GetBucketWebsite(ctx context.Context, bucketName string) (*s3.GetBucketWebsiteOutput, error) {
	input := &s3.GetBucketWebsiteInput{
		Bucket: aws.String(bucketName),
	}
	
	result, err := c.s3Client.GetBucketWebsite(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket website configuration for %s: %w", bucketName, err)
	}
	
	return result, nil
}
