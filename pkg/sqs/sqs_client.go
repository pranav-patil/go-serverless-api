package helper

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/pkg/errors"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/rs/zerolog/log"
)

type sqsAPI struct {
	SQS *sqs.Client
}

type SQSClient interface {
	SendMessage(queueName, message string) (*sqs.SendMessageOutput, error)
	SendBatchMessages(queueURL string, messages []string) error
	ReceiveMessages(queueURL string, maxRecvNum int) ([]types.Message, error)
	DeleteMessage(queueURL, receiptHandle string) error
}

func NewSQSClient() (SQSClient, error) {
	sqsAPI := new(sqsAPI)

	var cfg aws.Config
	var err error

	if env.IsLocalOrTestEnv() {
		cfg, err = mockutil.GetLocalStackConfig()
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		log.Error().Msgf("SNS LoadDefaultConfig Error: %v", err.Error())
		return sqsAPI, err
	}

	sqsClient := sqs.NewFromConfig(cfg)
	sqsAPI.SQS = sqsClient
	return sqsAPI, nil
}

func (api *sqsAPI) SendMessage(queueURL, message string) (*sqs.SendMessageOutput, error) {
	return api.SQS.SendMessage(
		context.TODO(),
		&sqs.SendMessageInput{
			MessageBody: &message,
			QueueUrl:    aws.String(queueURL),
		},
	)
}

func (api *sqsAPI) SendBatchMessages(queueURL string, messages []string) error {
	var messageEntries []types.SendMessageBatchRequestEntry

	for i, message := range messages {
		entry := types.SendMessageBatchRequestEntry{
			Id:          aws.String(strconv.Itoa(i)),
			MessageBody: aws.String(message),
		}
		messageEntries = append(messageEntries, entry)
	}

	input := &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(queueURL),
		Entries:  messageEntries,
	}

	_, err := api.SQS.SendMessageBatch(context.TODO(), input)
	return err
}

func (api *sqsAPI) ReceiveMessages(queueURL string, maxRecvNum int) ([]types.Message, error) {
	output, err := api.SQS.ReceiveMessage(
		context.TODO(),
		&sqs.ReceiveMessageInput{
			QueueUrl:              aws.String(queueURL),
			MaxNumberOfMessages:   int32(maxRecvNum),
			MessageAttributeNames: []string{"All"},
		},
	)

	if err != nil {
		return []types.Message{}, errors.Wrap(err, "SQS ReceiveMessage error")
	}

	return output.Messages, nil
}

func (api *sqsAPI) DeleteMessage(queueURL, receiptHandle string) error {
	_, err := api.SQS.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	return err
}
