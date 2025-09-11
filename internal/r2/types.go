package r2

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// BucketInfo represents detailed information about a bucket
type BucketInfo struct {
	Name         string             `json:"name"`
	CreationDate time.Time          `json:"creation_date"`
	Region       string             `json:"region,omitempty"`
	Location     string             `json:"location,omitempty"`
	Policy       *BucketPolicyInfo  `json:"policy,omitempty"`
	Website      *BucketWebsiteInfo `json:"website,omitempty"`
	Error        string             `json:"error,omitempty"`
}

// BucketPolicyInfo represents bucket policy information
type BucketPolicyInfo struct {
	HasPolicy  bool   `json:"has_policy"`
	PolicySize int64  `json:"policy_size,omitempty"`
	Error      string `json:"error,omitempty"`
}

// BucketWebsiteInfo represents bucket website configuration information
type BucketWebsiteInfo struct {
	Enabled             bool   `json:"enabled"`
	IndexDocument       string `json:"index_document,omitempty"`
	ErrorDocument       string `json:"error_document,omitempty"`
	RedirectAllRequests string `json:"redirect_all_requests,omitempty"`
	Error               string `json:"error,omitempty"`
}

// NewBucketInfoFromAWS creates a BucketInfo from AWS SDK Bucket type
func NewBucketInfoFromAWS(bucket types.Bucket) *BucketInfo {
	info := &BucketInfo{
		Name: *bucket.Name,
	}

	if bucket.CreationDate != nil {
		info.CreationDate = *bucket.CreationDate
	}

	return info
}

// NewBucketPolicyInfoFromAWS creates a BucketPolicyInfo from AWS SDK GetBucketPolicyOutput
func NewBucketPolicyInfoFromAWS(output *s3.GetBucketPolicyOutput) *BucketPolicyInfo {
	if output == nil {
		return &BucketPolicyInfo{
			HasPolicy: false,
		}
	}

	info := &BucketPolicyInfo{
		HasPolicy: true,
	}

	if output.Policy != nil {
		info.PolicySize = int64(len(*output.Policy))
	}

	return info
}

// NewBucketPolicyInfoWithError creates a BucketPolicyInfo with error information
func NewBucketPolicyInfoWithError(err error) *BucketPolicyInfo {
	return &BucketPolicyInfo{
		HasPolicy: false,
		Error:     err.Error(),
	}
}

// NewBucketWebsiteInfoFromAWS creates a BucketWebsiteInfo from AWS SDK GetBucketWebsiteOutput
func NewBucketWebsiteInfoFromAWS(output *s3.GetBucketWebsiteOutput) *BucketWebsiteInfo {
	if output == nil {
		return &BucketWebsiteInfo{
			Enabled: false,
		}
	}

	info := &BucketWebsiteInfo{
		Enabled: true,
	}

	if output.IndexDocument != nil && output.IndexDocument.Suffix != nil {
		info.IndexDocument = *output.IndexDocument.Suffix
	}

	if output.ErrorDocument != nil && output.ErrorDocument.Key != nil {
		info.ErrorDocument = *output.ErrorDocument.Key
	}

	if output.RedirectAllRequestsTo != nil && output.RedirectAllRequestsTo.HostName != nil {
		info.RedirectAllRequests = *output.RedirectAllRequestsTo.HostName
	}

	return info
}

// NewBucketWebsiteInfoWithError creates a BucketWebsiteInfo with error information
func NewBucketWebsiteInfoWithError(err error) *BucketWebsiteInfo {
	return &BucketWebsiteInfo{
		Enabled: false,
		Error:   err.Error(),
	}
}

// SetError sets an error message on the BucketInfo
func (b *BucketInfo) SetError(err error) {
	b.Error = err.Error()
}

// SetRegion sets the region information on the BucketInfo
func (b *BucketInfo) SetRegion(region string) {
	b.Region = region
	b.Location = region
}

// SetPolicy sets the policy information on the BucketInfo
func (b *BucketInfo) SetPolicy(policy *BucketPolicyInfo) {
	b.Policy = policy
}

// SetWebsite sets the website information on the BucketInfo
func (b *BucketInfo) SetWebsite(website *BucketWebsiteInfo) {
	b.Website = website
}
