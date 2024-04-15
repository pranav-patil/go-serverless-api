package sns

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sns"
)

//go:generate mockgen -destination mocks/aws_sns_client_mock.go -package mocks . AWSSNSClient

type AWSSNSClient interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}
