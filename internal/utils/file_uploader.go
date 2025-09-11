package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
)

// ProgressCallback 定义进度回调类型
type ProgressCallback func(uploaded, total int64, percentage float64)

// FileUploader 接口定义上传器的核心功能
type FileUploader interface {
	// UploadFile 上传单个文件
	UploadFile(ctx context.Context, localPath, remotePath string, options *UploadOptions) error

	// UploadFileWithProgress 上传文件并提供进度回调
	UploadFileWithProgress(ctx context.Context, localPath, remotePath string, options *UploadOptions, callback ProgressCallback) error

	// CheckFileExists 检查远程文件是否存在
	CheckFileExists(ctx context.Context, remotePath string) (bool, error)
}

// UploadOptions 上传选项
type UploadOptions struct {
	Overwrite        bool
	PublicAccess     bool
	ContentType      string
	CompressionLevel string
}

// S3ClientInterface 定义 S3 客户端接口，便于测试
type S3ClientInterface interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

// R2ClientInterface 定义 R2 客户端接口，便于测试
type R2ClientInterface interface {
	GetS3Client() interface{}
}

// uploadError 包装上传相关的错误
type uploadError struct {
	operation string
	path      string
	err       error
}

func (e *uploadError) Error() string {
	return fmt.Sprintf("upload %s failed for %s: %v", e.operation, e.path, e.err)
}

func (e *uploadError) Unwrap() error {
	return e.err
}

// fileUploader 是 FileUploader 接口的具体实现
type fileUploader struct {
	r2Client   R2ClientInterface
	s3Client   S3ClientInterface
	config     *config.Config
	bucketName string
}

// NewFileUploader 创建新的文件上传器
func NewFileUploader(client R2ClientInterface, cfg *config.Config, bucketName string) FileUploader {
	s3Client := extractS3Client(client)

	return &fileUploader{
		r2Client:   client,
		s3Client:   s3Client,
		config:     cfg,
		bucketName: bucketName,
	}
}

// extractS3Client 从 R2 客户端中提取 S3 客户端
func extractS3Client(client R2ClientInterface) S3ClientInterface {
	if client == nil {
		return nil
	}

	s3ClientRaw := client.GetS3Client()
	if s3ClientRaw == nil {
		return nil
	}

	// 尝试类型断言
	if s3c, ok := s3ClientRaw.(S3ClientInterface); ok {
		return s3c
	}

	if s3c, ok := s3ClientRaw.(*s3.Client); ok {
		return s3c
	}

	return nil
}

// UploadFile 实现 FileUploader 接口
func (fu *fileUploader) UploadFile(ctx context.Context, localPath, remotePath string, options *UploadOptions) error {
	return fu.UploadFileWithProgress(ctx, localPath, remotePath, options, nil)
}

// UploadFileWithProgress 实现带进度回调的文件上传
func (fu *fileUploader) UploadFileWithProgress(ctx context.Context, localPath, remotePath string, options *UploadOptions, callback ProgressCallback) error {
	if options == nil {
		options = &UploadOptions{}
	}

	// 打开并验证本地文件
	file, fileInfo, err := fu.openAndValidateFile(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 检查远程文件冲突
	if err := fu.checkRemoteFileConflict(ctx, remotePath, options.Overwrite); err != nil {
		return err
	}

	fileSize := fileInfo.Size()

	// 确定内容类型
	contentType := fu.determineContentType(localPath, file, options.ContentType)

	// 准备上传体
	uploadBody, err := fu.prepareUploadBody(file, fileSize, callback)
	if err != nil {
		return err
	}

	// 执行上传
	return fu.performUpload(ctx, localPath, remotePath, uploadBody, contentType, options.PublicAccess)
}

// CheckFileExists 检查远程文件是否存在
func (fu *fileUploader) CheckFileExists(ctx context.Context, remotePath string) (bool, error) {
	_, err := fu.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(fu.bucketName),
		Key:    aws.String(remotePath),
	})

	if err != nil {
		// 检查是否是"未找到"类型的错误
		var nsk *types.NoSuchKey
		var nf *types.NotFound
		if errors.As(err, &nsk) || errors.As(err, &nf) {
			return false, nil
		}

		// 也检查错误消息中是否包含 404 或 NotFound
		if strings.Contains(err.Error(), "StatusCode: 404") ||
			strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// openAndValidateFile 打开文件并获取文件信息
func (fu *fileUploader) openAndValidateFile(localPath string) (*os.File, os.FileInfo, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, nil, &uploadError{
			operation: "open file",
			path:      localPath,
			err:       err,
		}
	}

	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, &uploadError{
			operation: "get file info",
			path:      localPath,
			err:       err,
		}
	}

	return file, fileInfo, nil
}

// checkRemoteFileConflict 检查远程文件冲突
func (fu *fileUploader) checkRemoteFileConflict(ctx context.Context, remotePath string, overwrite bool) error {
	if overwrite {
		return nil // 允许覆盖，跳过检查
	}

	exists, err := fu.CheckFileExists(ctx, remotePath)
	if err != nil {
		return &uploadError{
			operation: "check remote file",
			path:      remotePath,
			err:       err,
		}
	}

	if exists {
		return &uploadError{
			operation: "check file conflict",
			path:      remotePath,
			err:       errors.New("file already exists and overwrite is disabled"),
		}
	}

	return nil
}

// prepareUploadBody 准备上传体，包括进度跟踪
func (fu *fileUploader) prepareUploadBody(file *os.File, fileSize int64, callback ProgressCallback) (io.Reader, error) {
	// 重置文件指针到开头
	if _, err := file.Seek(0, 0); err != nil {
		return nil, &uploadError{
			operation: "seek file",
			path:      file.Name(),
			err:       err,
		}
	}

	var uploadBody io.Reader = file
	if callback != nil {
		uploadBody = &progressReader{
			reader:   file,
			total:    fileSize,
			callback: callback,
		}
	}

	return uploadBody, nil
}

// determineContentType 确定文件的内容类型
func (fu *fileUploader) determineContentType(localPath string, file *os.File, explicitType string) string {
	if explicitType != "" {
		return explicitType
	}

	// 保存当前文件位置
	currentPos, _ := file.Seek(0, 1)

	contentType, _ := DetectContentType(localPath, file)

	// 恢复文件位置
	file.Seek(currentPos, 0)

	return contentType
}

// performUpload 执行实际的上传操作
func (fu *fileUploader) performUpload(ctx context.Context, localPath, remotePath string, uploadBody io.Reader, contentType string, publicAccess bool) error {
	// 准备上传参数
	input := &s3.PutObjectInput{
		Bucket: aws.String(fu.bucketName),
		Key:    aws.String(remotePath),
		Body:   uploadBody,
	}

	// 设置内容类型
	if contentType != "" {
		input.ContentType = aws.String(contentType)
		logrus.Debugf("Setting content type: %s", contentType)
	}

	// 设置公共访问权限
	if publicAccess {
		input.ACL = "public-read"
		logrus.Debugf("Setting public access")
	}

	// 执行上传
	_, err := fu.s3Client.PutObject(ctx, input)
	if err != nil {
		return &uploadError{
			operation: "upload to S3",
			path:      localPath,
			err:       err,
		}
	}

	logrus.Infof("Successfully uploaded %s to %s", localPath, remotePath)
	return nil
}

// progressReader 包装 io.Reader 并提供进度回调
type progressReader struct {
	reader   io.Reader
	total    int64
	read     int64
	callback ProgressCallback
}

// Read 实现 io.Reader 接口并触发进度回调
func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)

	if n > 0 {
		pr.read += int64(n)
		if pr.callback != nil {
			percentage := float64(pr.read) / float64(pr.total) * 100
			if percentage > 100 {
				percentage = 100
			}
			pr.callback(pr.read, pr.total, percentage)
		}
	}

	return n, err
}

// Seek 实现 io.Seeker 接口（如果底层读取器支持）
func (pr *progressReader) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := pr.reader.(io.Seeker); ok {
		pos, err := seeker.Seek(offset, whence)
		if err == nil {
			pr.read = pos
		}
		return pos, err
	}
	return 0, fmt.Errorf("underlying reader does not support seeking")
}
