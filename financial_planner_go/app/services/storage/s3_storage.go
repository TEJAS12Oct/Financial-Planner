package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	Folder    string
}

type S3Storage struct {
	client *s3.Client
	cfg    S3Config
}

func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		endpoint := strings.TrimSpace(cfg.Endpoint)
		if endpoint != "" {
			o.BaseEndpoint = aws.String(strings.TrimRight(endpoint, "/"))
		}
	})
	return &S3Storage{client: client, cfg: cfg}, nil
}

func (s *S3Storage) DownloadPDF(ctx context.Context, key string) ([]byte, error) {
	cleanKey := strings.TrimPrefix(strings.TrimSpace(key), "/")
	if cleanKey == "" {
		return nil, fmt.Errorf("S3 object key is required")
	}

	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(cleanKey),
	})
	if err != nil {
		return nil, fmt.Errorf("download PDF from S3: %w", err)
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("read PDF from S3: %w", err)
	}

	return data, nil
}

func (s *S3Storage) UploadPDF(ctx context.Context, filename string, data []byte) (string, string, error) {
	key := path.Join(strings.Trim(s.cfg.Folder, "/"), filename)
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.cfg.Bucket), Key: aws.String(key), Body: bytes.NewReader(data),
		ContentType: aws.String("application/pdf"),
	})
	if err != nil {
		return "", "", fmt.Errorf("upload PDF to S3: %w", err)
	}
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.cfg.Bucket, s.cfg.Region, key)
	return url, key, nil
}
