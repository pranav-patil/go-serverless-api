package helper

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/rs/zerolog/log"
)

type cloudwatchAPI struct {
	CW *cloudwatchlogs.Client
}

type CloudWatchClient interface {
	CreateLogStream(groupName, streamName string) error
	PutLogEvents(groupName, streamName, nextSequenceToken, message string) (string, error)
}

func NewCloudWatchClient() (CloudWatchClient, error) {
	cloudwatchAPI := new(cloudwatchAPI)

	var cfg aws.Config
	var err error

	if env.IsLocalOrTestEnv() {
		cfg, err = mockutil.GetLocalStackConfig()
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		log.Error().Msgf("SNS LoadDefaultConfig Error: %v", err.Error())
		return cloudwatchAPI, err
	}

	cwClient := cloudwatchlogs.NewFromConfig(cfg)
	cloudwatchAPI.CW = cwClient
	return cloudwatchAPI, nil
}

func (api *cloudwatchAPI) CreateLogStream(groupName, streamName string) error {
	createIn := cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(groupName),
		LogStreamName: aws.String(streamName),
	}
	_, cerr := api.CW.CreateLogStream(context.TODO(), &createIn)
	return cerr
}

func (api *cloudwatchAPI) PutLogEvents(groupName, streamName, nextSequenceToken, message string) (string, error) {
	in := cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(groupName),
		LogStreamName: aws.String(streamName),
		LogEvents: []types.InputLogEvent{
			{
				Message:   aws.String(message),
				Timestamp: aws.Int64(time.Now().UnixMilli()),
			},
		},
	}

	var err error
	if nextSequenceToken == "" {
		nextSequenceToken, err = api.getNextSequenceToken(groupName, streamName)
		if err != nil {
			return "", err
		}
	}

	if nextSequenceToken != "" {
		in.SequenceToken = aws.String(nextSequenceToken)
	}

	output, err := api.CW.PutLogEvents(context.TODO(), &in)
	if err != nil {
		return "", err
	}

	return *output.NextSequenceToken, err
}

func (api *cloudwatchAPI) getNextSequenceToken(groupName, streamName string) (string, error) {
	describeIn := cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(groupName),
		LogStreamNamePrefix: aws.String(streamName),
	}
	out, err := api.CW.DescribeLogStreams(context.TODO(), &describeIn)

	if err != nil {
		return "", err
	}

	for _, ls := range out.LogStreams {
		if *ls.LogStreamName == streamName {
			if ls.UploadSequenceToken != nil {
				return *ls.UploadSequenceToken, nil
			} else {
				return "", nil
			}
		}
	}

	err = api.CreateLogStream(groupName, streamName)
	return "", err
}
