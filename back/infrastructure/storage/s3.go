package storage

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/finde-clip/finde-v2/back/internal/media"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type S3 struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}
type Settings struct {
	Endpoint, PublicEndpoint, AccessKey, SecretKey, Bucket, Region string
	UseSSL                                                         bool
}

func New(ctx context.Context, s Settings) (*S3, error) {
	scheme := "http"
	if s.UseSSL {
		scheme = "https"
	}
	endpoint := scheme + "://" + strings.TrimPrefix(strings.TrimPrefix(s.Endpoint, "http://"), "https://")
	_, e := url.Parse(endpoint)
	if e != nil {
		return nil, e
	}
	// PutObject receives browser streams. MinIO over HTTP cannot replay them for optional SDK checksums.
	cfg, e := config.LoadDefaultConfig(ctx, config.WithRegion(s.Region), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s.AccessKey, s.SecretKey, "")), config.WithBaseEndpoint(endpoint), config.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired))
	if e != nil {
		return nil, e
	}
	c := s3.NewFromConfig(cfg, func(o *s3.Options) { o.UsePathStyle = true })
	publicCfg, e := config.LoadDefaultConfig(ctx, config.WithRegion(s.Region), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s.AccessKey, s.SecretKey, "")), config.WithBaseEndpoint(s.PublicEndpoint), config.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired))
	if e != nil {
		return nil, e
	}
	publicClient := s3.NewFromConfig(publicCfg, func(o *s3.Options) { o.UsePathStyle = true })
	return &S3{c, s3.NewPresignClient(publicClient), s.Bucket}, nil
}
func (s *S3) EnsureBucket(ctx context.Context) error {
	_, e := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if e == nil {
		return nil
	}
	_, e = s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)})
	return e
}
func (s *S3) Put(ctx context.Context, key string, body io.Reader, mime string) error {
	if _, ok := body.(io.ReadSeeker); !ok {
		// The AWS signer needs to replay HTTP payloads; browser downloads are forward-only streams.
		file, err := os.CreateTemp("", "finde-s3-upload-*")
		if err != nil {
			return err
		}
		defer os.Remove(file.Name())
		defer file.Close()
		if _, err = io.Copy(file, body); err != nil {
			return err
		}
		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return err
		}
		body = file
	}
	_, e := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), Body: body, ContentType: aws.String(mime)})
	return e
}
func (s *S3) Get(ctx context.Context, key string) (media.Object, error) {
	r, e := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if e != nil {
		return media.Object{}, e
	}
	return media.Object{Key: key, ContentType: aws.ToString(r.ContentType), Size: aws.ToInt64(r.ContentLength), Body: r.Body}, nil
}
func (s *S3) Delete(ctx context.Context, key string) error {
	_, e := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return e
}
func (s *S3) Exists(ctx context.Context, key string) (bool, error) {
	_, e := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if e == nil {
		return true, nil
	}
	var apiErr smithy.APIError
	if errors.As(e, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
		return false, nil
	}
	var responseErr *smithyhttp.ResponseError
	if errors.As(e, &responseErr) && responseErr.HTTPStatusCode() == http.StatusNotFound {
		return false, nil
	}
	return false, e
}
func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	r, e := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}, s3.WithPresignExpires(ttl))
	if e != nil {
		return "", e
	}
	return r.URL, nil
}
