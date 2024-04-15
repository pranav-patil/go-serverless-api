package stepfunc

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/rs/zerolog/log"
)

//go:generate mockgen -destination mocks/sfn_client_mock.go -package mocks . StepFuncClient

type StepFuncClient interface {
	StartExecution(stateMachineArn, executionName string, initialState interface{}) error
	SendTaskSuccess(taskToken, message string) error
	SendTaskFailure(taskToken, errMessage, errCause string) error
}

type sfnAPI struct {
	Sfn AWSStepFuncClient
}

func NewStepFunctionClient() (StepFuncClient, error) {
	stepFuncAPI := new(sfnAPI)

	var cfg aws.Config
	var err error

	if env.IsLocalOrTestEnv() {
		cfg, err = mockutil.GetLocalStackConfig()
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		log.Error().Msgf("S3 LoadDefaultConfig Error: %v", err.Error())
		return stepFuncAPI, err
	}

	stepFuncAPI.Sfn = sfn.NewFromConfig(cfg)
	return stepFuncAPI, nil
}

func (api *sfnAPI) StartExecution(stateMachineArn, executionName string, initialState interface{}) error {
	initialStateAsBytes, _ := json.Marshal(initialState)
	initialStateAsString := string(initialStateAsBytes)

	input := &sfn.StartExecutionInput{
		StateMachineArn: aws.String(stateMachineArn),
		Input:           aws.String(initialStateAsString),
		Name:            aws.String(executionName),
	}

	_, err := api.Sfn.StartExecution(context.TODO(), input)
	if err != nil {
		log.Error().Msgf("StartExecution Error: %v", err.Error())
	}

	return err
}

func (api *sfnAPI) SendTaskSuccess(taskToken, message string) error {
	sendTaskSuccessIn := &sfn.SendTaskSuccessInput{
		Output:    aws.String(message),
		TaskToken: aws.String(taskToken),
	}

	_, err := api.Sfn.SendTaskSuccess(context.TODO(), sendTaskSuccessIn)
	if err != nil {
		log.Error().Msgf("SendTaskSuccess Error: %v", err.Error())
	}

	return err
}

func (api *sfnAPI) SendTaskFailure(taskToken, errMessage, errCause string) error {
	sendTaskFailureIn := &sfn.SendTaskFailureInput{
		TaskToken: aws.String(taskToken),
		Error:     aws.String(errMessage),
		Cause:     aws.String(errCause),
	}
	_, err := api.Sfn.SendTaskFailure(context.TODO(), sendTaskFailureIn)
	if err != nil {
		log.Error().Msgf("SendTaskFailure Error: %v", err.Error())
	}

	return err
}
