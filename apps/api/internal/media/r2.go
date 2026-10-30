package media

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const (
	uploadURLLifetime = 15 * time.Minute
	viewURLLifetime   = time.Hour
)

var ErrObjectMissing = errors.New("object not found in storage")

type R2Config struct {
	AccountID       string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

func (c R2Config) Complete() bool {
	return c.AccountID != "" && c.Bucket != "" && c.AccessKeyID != "" && c.SecretAccessKey != ""
}

// R2 talks to Cloudflare R2 through its S3-compatible API. Browsers upload and
// play files with presigned URLs, so video bytes never pass through the Go API.
type R2 struct {
	bucket  string
	client  *s3.Client
	presign *s3.PresignClient
}

func NewR2(cfg R2Config) *R2 {
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		// Newer SDKs add CRC checksums to uploads by default; a browser PUT to a
		// presigned URL cannot send them, so only checksum when an API requires it.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
	return &R2{bucket: cfg.Bucket, client: client, presign: s3.NewPresignClient(client)}
}

// UploadURL signs Content-Type and Content-Length, so the browser must PUT
// exactly the declared file.
func (r *R2) UploadURL(ctx context.Context, key, contentType string, size int64) (string, error) {
	req, err := r.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}, s3.WithPresignExpires(uploadURLLifetime))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (r *R2) ViewURL(ctx context.Context, key string) (string, error) {
	req, err := r.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(viewURLLifetime))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// DownloadURL is a view URL that also makes R2 answer with
// Content-Disposition: attachment, so phones save the file instead of playing it.
// iOS Safari and Android Chrome ignore <a download> for cross-origin links; this
// header is what actually triggers the download there.
func (r *R2) DownloadURL(ctx context.Context, key, filename string) (string, error) {
	req, err := r.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(r.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(AttachmentDisposition(filename)),
	}, s3.WithPresignExpires(viewURLLifetime))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// AttachmentDisposition keeps the original (possibly Thai) filename: an ASCII
// fallback for old clients plus the RFC 6266 UTF-8 form that browsers prefer.
func AttachmentDisposition(filename string) string {
	fallback := strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, filename)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, fallback, url.PathEscape(filename))
}

// Stat returns the stored size and content type of an uploaded object.
func (r *R2) Stat(ctx context.Context, key string) (int64, string, error) {
	out, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(r.bucket), Key: aws.String(key)})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
			return 0, "", ErrObjectMissing
		}
		return 0, "", err
	}
	return aws.ToInt64(out.ContentLength), aws.ToString(out.ContentType), nil
}

func (r *R2) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(r.bucket), Key: aws.String(key)})
	return err
}
