package utils

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sirupsen/logrus"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
)

type URLGenerator struct {
	s3Client   *s3.Client
	config     *config.Config
	bucketName string
}

func NewURLGenerator(s3Client *s3.Client, cfg *config.Config, bucketName string) *URLGenerator {
	return &URLGenerator{
		s3Client:   s3Client,
		config:     cfg,
		bucketName: bucketName,
	}
}

// SetBucketName updates the bucket name for URL generation
func (g *URLGenerator) SetBucketName(bucketName string) {
	logrus.Tracef("URLGenerator.SetBucketName: updating bucket from %s to %s", g.bucketName, bucketName)
	g.bucketName = bucketName
}

func (g *URLGenerator) GenerateFileURL(key string) (customURL string, presignedURL string, err error) {
	// Generate custom domain URL if configured for this bucket
	customDomain := g.config.GetCustomDomain(g.bucketName)
	if customDomain != "" {
		customURL = g.GenerateCustomDomainURL(key)
	}

	// Generate presigned URL
	presignedURL, err = g.GeneratePresignedURL(key)
	if err != nil {
		logrus.Errorf("Failed to generate presigned URL for %s: %v", key, err)
		return customURL, "", err
	}

	return customURL, presignedURL, nil
}

func (g *URLGenerator) GenerateCustomDomainURL(key string) string {
	// Get custom domain for this bucket
	domain := g.config.GetCustomDomain(g.bucketName)
	logrus.Tracef("URLGenerator.GenerateCustomDomainURL: bucket=%s, domain=%s", g.bucketName, domain)
	if domain == "" {
		return ""
	}

	// Clean up domain (remove protocol if present)
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	// URL encode the key properly
	encodedKey := url.PathEscape(key)

	return fmt.Sprintf("https://%s/%s", domain, encodedKey)
}

func (g *URLGenerator) GeneratePresignedURL(key string) (string, error) {
	presignClient := s3.NewPresignClient(g.s3Client)

	// Create presigned URL valid for 1 hour
	request, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(g.bucketName),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Hour
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign request: %w", err)
	}

	return request.URL, nil
}

func (g *URLGenerator) GetPreferredURL(key string) (string, error) {
	customURL, presignedURL, err := g.GenerateFileURL(key)
	if err != nil {
		return "", err
	}

	// Prefer custom domain URL if available
	if customURL != "" {
		return customURL, nil
	}

	return presignedURL, nil
}

func (g *URLGenerator) GenerateAllURLs(key string) (map[string]string, error) {
	customURL, presignedURL, err := g.GenerateFileURL(key)
	if err != nil {
		return nil, err
	}

	urls := make(map[string]string)

	if customURL != "" {
		urls["Custom Domain"] = customURL
	}

	if presignedURL != "" {
		urls["Presigned URL"] = presignedURL
	}

	return urls, nil
}

func formatURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Clean up the URL for better display
	if len(u.Path) > 50 {
		dir := path.Dir(u.Path)
		base := path.Base(u.Path)
		if len(base) > 30 {
			base = base[:27] + "..."
		}
		u.Path = dir + "/" + base
	}

	return u.String()
}
