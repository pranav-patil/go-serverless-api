package sns

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/rs/zerolog/log"
)

//go:generate mockgen -destination mocks/sns_client_mock.go -package mocks . SNSClient

type SNSClient interface {
	Publish(snsTopicARN, message string) error
	PublishWithAttributes(snsTopicARN, message string, attrs map[string]string) error
}

type snsAPI struct {
	SNS AWSSNSClient
}

func NewSNSClient() (SNSClient, error) {
	snsAPI := new(snsAPI)

	var cfg aws.Config
	var err error

	if env.IsLocalOrTestEnv() {
		cfg, err = mockutil.GetLocalStackConfig()
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		log.Error().Msgf("SNS LoadDefaultConfig Error: %v", err.Error())
		return snsAPI, err
	}

	snsClient := sns.NewFromConfig(cfg)
	snsAPI.SNS = snsClient
	return snsAPI, nil
}

func (api *snsAPI) Publish(snsTopicARN, message string) error {
	return api.PublishWithAttributes(snsTopicARN, message, nil)
}

func (api *snsAPI) PublishWithAttributes(snsTopicARN, message string, attrs map[string]string) error {
	attributes := map[string]types.MessageAttributeValue{}

	if len(attrs) > 0 {
		for key, value := range attrs {
			attributes[key] = types.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(value),
			}
		}
	}

	publishInput := &sns.PublishInput{
		TopicArn:          aws.String(snsTopicARN),
		MessageAttributes: attributes,
		Message:           aws.String(message),
	}

	result, err := api.SNS.Publish(context.TODO(), publishInput)
	if err != nil {
		log.Error().Msgf("SNS Error in sending message %s: %v", message, err.Error())
	}
	log.Debug().Msgf("Successfully Sent message id %s to TopicArn %s", *result.MessageId, snsTopicARN)

	return err
}
