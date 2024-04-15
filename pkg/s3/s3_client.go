package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/pranav-patil/go-serverless-api/pkg/util"
	"github.com/rs/zerolog/log"
)

//go:generate mockgen -destination mocks/s3_client_mock.go -package mocks . S3Client

type S3Client interface {
	CreateBucket(bucket, region string) error
	PutObject(bucket, key, contentType, encoding string, content *[]byte) error
	GetObject(bucket, key string) ([]byte, error)
	DeleteObject(bucket, key string) error
	DeleteObjectsWithPrefix(bucket, prefix string) error
	DeleteObjects(bucket string, objectKeys []string) error
	ObjectExists(bucket, key string) (bool, error)
	ListObjects(bucket, prefix string) ([]types.Object, error)
	DeleteBucket(bucket string) error
	CopyObject(sourceBucket, destinationBucket, key string) error
	NewSignedGetURL(bucket, key string, lifetimeSecs int64) (string, error)
}

const (
	GZip string = "gzip"
)

type s3Api struct {
	S3      AWSS3Client
	Presign AWSS3PresignClient
}

func NewS3Client() (S3Client, error) {
	s3Api := new(s3Api)

	var cfg aws.Config
	var err error
	usePathStyle := false

	if env.IsLocalOrTestEnv() {
		cfg, err = mockutil.GetLocalStackConfig()
		usePathStyle = true
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		log.Error().Msgf("S3 LoadDefaultConfig Error: %v", err.Error())
		return s3Api, err
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = usePathStyle
	})
	s3Api.S3 = s3Client
	s3Api.Presign = s3.NewPresignClient(s3Client)
	return s3Api, nil
}

func (api *s3Api) CreateBucket(bucket, region string) error {
	bucketInput := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		},
	}

	_, err := api.S3.CreateBucket(context.TODO(), bucketInput)
	if err != nil {
		log.Error().Msgf("S3 CreateBucket Error: %v", err.Error())
	}

	return err
}

func (api *s3Api) PutObject(bucket, key, contentType, encoding string, content *[]byte) error {
	if content == nil {
		return fmt.Errorf("put content is nil")
	}

	if encoding == GZip {
		response, err := util.Compress(string(*content))
		if err != nil {
			return err
		}
		content = &response
	}

	objectInput := &s3.PutObjectInput{
		Bucket:          aws.String(bucket),
		Key:             aws.String(key),
		Body:            bytes.NewReader(*content),
		ContentType:     aws.String(contentType),
		ContentEncoding: aws.String(encoding),
	}

	_, err := api.S3.PutObject(context.TODO(), objectInput)
	if err != nil {
		log.Error().Msgf("S3 PutObject Error: %v", err.Error())
	}

	return err
}

func (api *s3Api) GetObject(bucket, key string) ([]byte, error) {
	objectInput := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	objectOutput, err := api.S3.GetObject(context.TODO(), objectInput)
	if err != nil {
		log.Error().Msgf("S3 GetObject Error: %v", err.Error())
		return nil, err
	}

	defer func() {
		if cerr := objectOutput.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	bodyContent, err := io.ReadAll(objectOutput.Body)
	if err != nil {
		log.Error().Msgf("S3 IO ReadAll Error: %v", err.Error())
		return nil, err
	}

	if *objectOutput.ContentEncoding == GZip {
		response, err := util.Decompress(bodyContent)
		if err != nil {
			return nil, err
		}
		bodyContent = []byte(response)
	}

	return bodyContent, nil
}

func (api *s3Api) DeleteObject(bucket, key string) error {
	objectInput := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := api.S3.DeleteObject(context.TODO(), objectInput)
	if err != nil {
		log.Error().Msgf("S3 DeleteObject Error: %v", err.Error())
	}

	return err
}

func (api *s3Api) DeleteObjectsWithPrefix(bucket, prefix string) error {
	contents, err := api.ListObjects(bucket, prefix)
	if err != nil {
		return err
	}

	objectKeys := []string{}

	for _, v := range contents {
		objectKeys = append(objectKeys, *v.Key)
	}

	return api.DeleteObjects(bucket, objectKeys)
}

func (api *s3Api) DeleteObjects(bucket string, objectKeys []string) error {
	var objectIds []types.ObjectIdentifier
	for _, key := range objectKeys {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: aws.String(key)})
	}
	_, err := api.S3.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &types.Delete{Objects: objectIds},
	})
	if err != nil {
		log.Error().Msgf("S3 DeleteObjects Error for bucket %v: %v", bucket, err.Error())
	}
	return err
}

func (api *s3Api) ObjectExists(bucket, key string) (bool, error) {
	objectInput := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	if _, err := api.S3.HeadObject(context.TODO(), objectInput); err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return false, nil
		}
	}

	return true, nil
}

func (api *s3Api) ListObjects(bucket, prefix string) ([]types.Object, error) {
	result, err := api.S3.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	var contents []types.Object
	if err != nil {
		log.Error().Msgf("S3 ListObjects Error for bucket %v: %v", bucket, err.Error())
	} else {
		contents = result.Contents
	}
	return contents, err
}

func (api *s3Api) DeleteBucket(bucket string) error {
	bucketInput := &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	}

	_, err := api.S3.DeleteBucket(context.TODO(), bucketInput)
	if err != nil {
		log.Error().Msgf("S3 DeleteBucket Error: %v", err.Error())
	}

	return err
}

func (api *s3Api) CopyObject(sourceBucket, destinationBucket, key string) error {
	copyObjectInput := &s3.CopyObjectInput{
		Bucket:     aws.String(destinationBucket),
		CopySource: aws.String(fmt.Sprintf("%v/%v", sourceBucket, key)),
		Key:        aws.String(key),
	}

	_, err := api.S3.CopyObject(context.TODO(), copyObjectInput)
	if err != nil {
		log.Error().Msgf("S3 CopyObject Error: %v", err.Error())
	}

	return err
}

func (api *s3Api) NewSignedGetURL(bucket, key string, lifetimeSecs int64) (string, error) {
	request, err := api.Presign.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(lifetimeSecs * int64(time.Second))
	})
	if err != nil {
		log.Error().Msgf("S3 Presign GetURL Error for bucket %v, key %v: %v", bucket, key, err.Error())
		return "", err
	}
	return request.URL, err
}
