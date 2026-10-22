package storage

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// Part is one uploaded piece of a multipart object.
type Part struct {
	Number int32
	ETag   string
	Size   int64
}

// Store is the slice of S3 the app needs. Browsers talk to the bucket
// directly through presigned URLs, so file bytes never cross our API.
type Store interface {
	CreateMultipart(ctx context.Context, key, contentType string) (uploadID string, err error)
	PresignPart(ctx context.Context, key, uploadID string, part int32, ttl time.Duration) (string, error)
	ListParts(ctx context.Context, key, uploadID string) ([]Part, error)
	Complete(ctx context.Context, key, uploadID string, parts []Part) error
	Abort(ctx context.Context, key, uploadID string) error
	PresignDownload(ctx context.Context, key, filename string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

type Config struct {
	Endpoint       string // used by the API itself (e.g. http://minio:9000)
	PublicEndpoint string // embedded in presigned URLs for browsers; defaults to Endpoint
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	PathStyle      bool
}

type S3 struct {
	bucket  string
	client  *s3.Client
	presign *s3.PresignClient
}

func New(cfg Config) *S3 {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.PublicEndpoint == "" {
		cfg.PublicEndpoint = cfg.Endpoint
	}
	newClient := func(endpoint string) *s3.Client {
		return s3.New(s3.Options{
			Region:       cfg.Region,
			BaseEndpoint: aws.String(endpoint),
			UsePathStyle: cfg.PathStyle,
			Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
			// Arvan and MinIO do not implement the newer default checksums.
			RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
			ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
		})
	}
	return &S3{
		bucket:  cfg.Bucket,
		client:  newClient(cfg.Endpoint),
		presign: s3.NewPresignClient(newClient(cfg.PublicEndpoint)),
	}
}

// EnsureBucket creates the bucket if it is missing. Development convenience:
// production buckets are created by hand in the Arvan panel.
func (s *S3) EnsureBucket(ctx context.Context) error {
	_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &s.bucket})
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "BucketAlreadyOwnedByYou", "BucketAlreadyExists":
			return nil
		}
	}
	return err
}

func (s *S3) CreateMultipart(ctx context.Context, key, contentType string) (string, error) {
	out, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: &s.bucket, Key: &key, ContentType: &contentType,
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.UploadId), nil
}

func (s *S3) PresignPart(ctx context.Context, key, uploadID string, part int32, ttl time.Duration) (string, error) {
	req, err := s.presign.PresignUploadPart(ctx, &s3.UploadPartInput{
		Bucket: &s.bucket, Key: &key, UploadId: &uploadID, PartNumber: &part,
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (s *S3) ListParts(ctx context.Context, key, uploadID string) ([]Part, error) {
	var parts []Part
	var marker *string
	for {
		out, err := s.client.ListParts(ctx, &s3.ListPartsInput{
			Bucket: &s.bucket, Key: &key, UploadId: &uploadID, PartNumberMarker: marker,
		})
		if err != nil {
			return nil, err
		}
		for _, p := range out.Parts {
			parts = append(parts, Part{Number: aws.ToInt32(p.PartNumber), ETag: aws.ToString(p.ETag), Size: aws.ToInt64(p.Size)})
		}
		if !aws.ToBool(out.IsTruncated) {
			return parts, nil
		}
		marker = out.NextPartNumberMarker
	}
}

func (s *S3) Complete(ctx context.Context, key, uploadID string, parts []Part) error {
	completed := make([]types.CompletedPart, len(parts))
	for i, p := range parts {
		completed[i] = types.CompletedPart{PartNumber: aws.Int32(p.Number), ETag: aws.String(p.ETag)}
	}
	_, err := s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket: &s.bucket, Key: &key, UploadId: &uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{Parts: completed},
	})
	return err
}

func (s *S3) Abort(ctx context.Context, key, uploadID string) error {
	_, err := s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket: &s.bucket, Key: &key, UploadId: &uploadID,
	})
	return err
}

func (s *S3) PresignDownload(ctx context.Context, key, filename string, ttl time.Duration) (string, error) {
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	if disposition == "" {
		disposition = "attachment"
	}
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket, Key: &key, ResponseContentDisposition: &disposition,
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign download: %w", err)
	}
	return req.URL, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	return err
}
