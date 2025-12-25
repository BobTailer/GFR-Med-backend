package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	defaultBucketName = "patient-categories"
)

type MinIOClient struct {
	client     *minio.Client
	bucketName string
}

func NewMinIOClient() (*MinIOClient, error) {
	endpoint := getEnv("MINIO_ENDPOINT", "localhost")
	port := getEnv("MINIO_PORT", "9000")
	accessKeyID := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	secretAccessKey := getEnv("MINIO_SECRET_KEY", "minioadmin")
	useSSL := getEnv("MINIO_USE_SSL", "false") == "true"

	endpointURL := fmt.Sprintf("%s:%s", endpoint, port)

	client, err := minio.New(endpointURL, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	minioClient := &MinIOClient{
		client:     client,
		bucketName: defaultBucketName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, defaultBucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, defaultBucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Created MinIO bucket: %s", defaultBucketName)
		
		publicPolicy := `{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::patient-categories/*"]
			}]
		}`
		err = client.SetBucketPolicy(ctx, defaultBucketName, publicPolicy)
		if err != nil {
			log.Printf("Warning: could not set bucket policy for public access: %v", err)
		} else {
			log.Printf("Set public read access for bucket: %s", defaultBucketName)
		}
	}

	log.Printf("MinIO client initialized successfully, bucket: %s", defaultBucketName)
	return minioClient, nil
}

func (m *MinIOClient) UploadFile(ctx context.Context, objectKey string, filePath string, contentType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileStat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}

	_, err = m.client.PutObject(ctx, m.bucketName, objectKey, file, fileStat.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	log.Printf("Uploaded file to MinIO: %s -> %s/%s", filePath, m.bucketName, objectKey)
	return nil
}

func (m *MinIOClient) UploadFromReader(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucketName, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload from reader: %w", err)
	}

	log.Printf("Uploaded data to MinIO: %s/%s", m.bucketName, objectKey)
	return nil
}

func (m *MinIOClient) GetObject(ctx context.Context, objectKey string) (*minio.Object, error) {
	obj, err := m.client.GetObject(ctx, m.bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return obj, nil
}

func (m *MinIOClient) StatObject(ctx context.Context, objectKey string) (*minio.ObjectInfo, error) {
	info, err := m.client.StatObject(ctx, m.bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}
	return &info, nil
}

func (m *MinIOClient) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	url, err := m.client.PresignedGetObject(ctx, m.bucketName, objectKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

func (m *MinIOClient) GetPublicURL(objectKey string) string {
	publicURL := getEnv("MINIO_PUBLIC_URL", "")
	if publicURL != "" {
		publicURL = strings.TrimSuffix(publicURL, "/")
		return fmt.Sprintf("%s/%s/%s", publicURL, m.bucketName, objectKey)
	}
	
	endpoint := getEnv("MINIO_ENDPOINT", "localhost")
	port := getEnv("MINIO_PORT", "9000")
	useSSL := getEnv("MINIO_USE_SSL", "false") == "true"
	
	protocol := "http"
	if useSSL {
		protocol = "https"
	}
	
	return fmt.Sprintf("%s://%s:%s/%s/%s", protocol, endpoint, port, m.bucketName, objectKey)
}

func (m *MinIOClient) ObjectExists(ctx context.Context, objectKey string) (bool, error) {
	_, err := m.client.StatObject(ctx, m.bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}
	return true, nil
}

func (m *MinIOClient) DeleteObject(ctx context.Context, objectKey string) error {
	err := m.client.RemoveObject(ctx, m.bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	log.Printf("Deleted object from MinIO: %s/%s", m.bucketName, objectKey)
	return nil
}

func GenerateObjectKey(filename string) string {
	parts := strings.Split(filename, "/")
	filename = parts[len(parts)-1]
	filename = strings.ToLower(filename)
	filename = strings.ReplaceAll(filename, " ", "_")
	
	return filename
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

