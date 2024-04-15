package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/rs/zerolog/log"
)

type kinesisAPI struct {
	KINESIS *kinesis.Client
}

type KinesisClient interface {
	PutRecord(streamName, accessKeyId, secretAccessKey, sessionToken string, data []byte) (*kinesis.PutRecordOutput, error)
	PutRecords(streamName, accessKeyId, secretAccessKey, sessionToken string,
		recordsRequest []types.PutRecordsRequestEntry) (*kinesis.PutRecordsOutput, error)
}

func NewKinesisClient() (KinesisClient, error) {
	kinesisAPI := new(kinesisAPI)

	var cfg aws.Config
	var err error

	if env.IsLocalOrTestEnv() {
		cfg, err = mockutil.GetLocalStackConfig()
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		log.Error().Msgf("SNS LoadDefaultConfig Error: %v", err.Error())
		return kinesisAPI, err
	}

	kinesisClient := kinesis.NewFromConfig(cfg)
	kinesisAPI.KINESIS = kinesisClient
	return kinesisAPI, nil
}

func (api *kinesisAPI) PutRecord(streamName, accessKeyId, secretAccessKey, sessionToken string,
	data []byte) (*kinesis.PutRecordOutput, error) {
	partitionKey := fmt.Sprint(time.Now().UnixMilli())

	return api.KINESIS.PutRecord(
		context.TODO(),
		&kinesis.PutRecordInput{
			Data:         data,
			PartitionKey: &partitionKey,
			StreamName:   aws.String(streamName),
		},
		func(o *kinesis.Options) {
			o.Credentials = credentials.NewStaticCredentialsProvider(accessKeyId, secretAccessKey, sessionToken)
		},
	)
}

func (api *kinesisAPI) PutRecords(streamName, accessKeyId, secretAccessKey, sessionToken string,
	records []types.PutRecordsRequestEntry) (*kinesis.PutRecordsOutput, error) {
	return api.KINESIS.PutRecords(
		context.TODO(),
		&kinesis.PutRecordsInput{
			Records:    records,
			StreamName: aws.String(streamName),
		},
		func(o *kinesis.Options) {
			o.Credentials = credentials.NewStaticCredentialsProvider(accessKeyId, secretAccessKey, sessionToken)
		},
	)
}
