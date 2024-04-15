package s3

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang/mock/gomock"

	"github.com/pranav-patil/go-serverless-api/pkg/s3/mocks"
	"github.com/pranav-patil/go-serverless-api/pkg/util"
	"github.com/stretchr/testify/suite"
)

var (
	s3BucketName = "test-bucket-name"
	anyS3Key     = "AnyS3key"
)

type S3ClientTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func TestS3ClientSuite(t *testing.T) {
	suite.Run(t, new(S3ClientTestSuite))
}

func (s *S3ClientTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
}

func (s *S3ClientTestSuite) SetupTest() {
	s.T().Log("SetupTest..")
}

func (s *S3ClientTestSuite) TestGetObject() {
	mockS3Client := mocks.NewMockAWSS3Client(s.ctrl)

	ctx := context.TODO()
	input := s3.GetObjectInput{Bucket: &s3BucketName, Key: &anyS3Key}

	payloadContent := "test,R\ntest2,R\ntest3,G"
	compressed, err := util.Compress(payloadContent)
	s.Nil(err)

	mockS3Client.EXPECT().GetObject(ctx, &input).Return(&s3.GetObjectOutput{
		Body:            io.NopCloser(strings.NewReader(string(compressed))),
		ContentEncoding: aws.String(GZip),
	}, nil).Times(1)

	api := s3Api{S3: mockS3Client}
	content, err := api.GetObject(s3BucketName, anyS3Key)
	if err != nil {
		s.T().Errorf("Expected no error, but got: '%v'", err)
	}

	s.Equal(payloadContent, string(content))
}

func (s *S3ClientTestSuite) TestDeleteObject() {
	mockS3Client := mocks.NewMockAWSS3Client(s.ctrl)

	ctx := context.TODO()
	input := s3.DeleteObjectInput{Bucket: &s3BucketName, Key: &anyS3Key}

	mockS3Client.EXPECT().DeleteObject(ctx, &input).Return(&s3.DeleteObjectOutput{}, nil).Times(1)

	api := s3Api{S3: mockS3Client}
	err := api.DeleteObject(s3BucketName, anyS3Key)
	if err != nil {
		s.T().Errorf("Expected no error, but got: '%v'", err)
	}

	s.NoError(err)
}

func (s *S3ClientTestSuite) TestPutObject() {
	mockS3Client := mocks.NewMockAWSS3Client(s.ctrl)

	ctx := context.TODO()

	content := "Test file content"
	payloadContent := []byte(content)
	compressed, err := util.Compress(content)
	s.Nil(err)

	mockS3Client.EXPECT().PutObject(ctx, &s3.PutObjectInput{
		Bucket:          aws.String(s3BucketName),
		Key:             aws.String(anyS3Key),
		Body:            bytes.NewReader(compressed),
		ContentType:     aws.String("content-type"),
		ContentEncoding: aws.String("gzip"),
	}).Return(&s3.PutObjectOutput{}, nil).Times(1)

	api := s3Api{S3: mockS3Client}
	err = api.PutObject(s3BucketName, anyS3Key, "content-type", "gzip", &payloadContent)
	if err != nil {
		s.T().Errorf("Expected no error, but got: '%v'", err)
	}

	s.NoError(err)
}
