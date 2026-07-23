package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- AWS Signature Version 2 requires HMAC-SHA1.
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ddp0_grader/app/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// ThumbnailStorage keeps object storage isolated from HTTP handlers and makes
// it possible to use private S3 buckets through short-lived signed URLs.
type ThumbnailStorage interface {
	Upload(ctx context.Context, problemID string, content []byte, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
	PresignedURL(ctx context.Context, key string) (string, error)
}

type S3ThumbnailStorage struct {
	bucket           string
	endpoint         string
	accessKey        string
	secretKey        string
	signatureVersion string
	client           *s3.Client
	presign          *s3.PresignClient
}

func NewS3ThumbnailStorage(ctx context.Context) (*S3ThumbnailStorage, error) {
	accessKey := strings.TrimSpace(config.GetEnv("s3_access_key_id"))
	secretKey := strings.TrimSpace(config.GetEnv("s3_secret_access_key"))
	endpoint := strings.TrimSpace(config.GetEnv("s3_endpoint_url"))
	bucket := strings.TrimSpace(config.GetEnv("s3_bucket_name"))
	if accessKey == "" || secretKey == "" || endpoint == "" || bucket == "" {
		return nil, fmt.Errorf("S3 thumbnail storage is not configured")
	}
	region := strings.TrimSpace(config.GetEnv("s3_region"))
	if region == "" {
		region = "us-east-1"
	}
	signatureVersion := strings.ToLower(strings.TrimSpace(config.GetEnv("s3_signature_version")))
	if signatureVersion == "" {
		signatureVersion = "v4"
	}
	if signatureVersion == "v2" {
		return &S3ThumbnailStorage{
			bucket:           bucket,
			endpoint:         strings.TrimRight(endpoint, "/"),
			accessKey:        accessKey,
			secretKey:        secretKey,
			signatureVersion: signatureVersion,
		}, nil
	}
	if signatureVersion != "v4" {
		return nil, fmt.Errorf("unsupported S3 signature version %q", signatureVersion)
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, resolvedRegion string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: endpoint, HostnameImmutable: true, SigningRegion: resolvedRegion}, nil
		})),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("configure S3 client: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) { options.UsePathStyle = true })
	return &S3ThumbnailStorage{bucket: bucket, signatureVersion: signatureVersion, client: client, presign: s3.NewPresignClient(client)}, nil
}

func (storage *S3ThumbnailStorage) Upload(ctx context.Context, problemID string, content []byte, contentType string) (string, error) {
	key := fmt.Sprintf("problem-thumbnails/%s/%s%s", problemID, uuid.NewString(), extensionForContentType(contentType))
	if storage.signatureVersion == "v2" {
		return key, storage.requestV2(ctx, http.MethodPut, key, contentType, bytes.NewReader(content))
	}
	_, err := storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(storage.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(content),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("private, max-age=86400"),
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (storage *S3ThumbnailStorage) Delete(ctx context.Context, key string) error {
	if storage.signatureVersion == "v2" {
		return storage.requestV2(ctx, http.MethodDelete, key, "", nil)
	}
	_, err := storage.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(storage.bucket), Key: aws.String(key)})
	return err
}

func (storage *S3ThumbnailStorage) PresignedURL(ctx context.Context, key string) (string, error) {
	if storage.signatureVersion == "v2" {
		return storage.presignedURLV2(key), nil
	}
	request, err := storage.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(storage.bucket), Key: aws.String(key)}, func(options *s3.PresignOptions) {
		options.Expires = 15 * time.Minute
	})
	if err != nil {
		return "", err
	}
	return request.URL, nil
}

func (storage *S3ThumbnailStorage) requestV2(ctx context.Context, method, key, contentType string, body io.Reader) error {
	request, err := http.NewRequestWithContext(ctx, method, storage.objectURL(key), body)
	if err != nil {
		return err
	}
	date := time.Now().UTC().Format(http.TimeFormat)
	request.Header.Set("Date", date)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.Header.Set("Authorization", "AWS "+storage.accessKey+":"+storage.signatureV2(method, contentType, date, storage.canonicalResource(key)))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	message, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
	return fmt.Errorf("S3 %s failed with status %d: %s", method, response.StatusCode, strings.TrimSpace(string(message)))
}

func (storage *S3ThumbnailStorage) presignedURLV2(key string) string {
	expires := time.Now().Add(15 * time.Minute).Unix()
	signature := storage.signatureV2(http.MethodGet, "", fmt.Sprintf("%d", expires), storage.canonicalResource(key))
	query := url.Values{"AWSAccessKeyId": {storage.accessKey}, "Expires": {fmt.Sprintf("%d", expires)}, "Signature": {signature}}
	return storage.objectURL(key) + "?" + query.Encode()
}

func (storage *S3ThumbnailStorage) signatureV2(method, contentType, date, resource string) string {
	stringToSign := strings.Join([]string{method, "", contentType, date, resource}, "\n")
	mac := hmac.New(sha1.New, []byte(storage.secretKey))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (storage *S3ThumbnailStorage) canonicalResource(key string) string {
	return "/" + storage.bucket + "/" + key
}

func (storage *S3ThumbnailStorage) objectURL(key string) string {
	segments := strings.Split(key, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return storage.endpoint + "/" + url.PathEscape(storage.bucket) + "/" + strings.Join(segments, "/")
}

func extensionForContentType(contentType string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ".gif"
	}
}
