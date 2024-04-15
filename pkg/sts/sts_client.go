package helper

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/rs/zerolog/log"
)

type stsAPI struct {
	STS *sts.Client
}

type STSClient interface {
	AssumeRole(roleArn, roleSessionName string, duration int32) (*sts.AssumeRoleOutput, error)
}

func NewSTSClient() (STSClient, error) {
	stsAPI := new(stsAPI)

	var cfg aws.Config
	var err error

	if env.IsLocalOrTestEnv() {
		cfg, err = mockutil.GetLocalStackConfig()
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		log.Error().Msgf("SNS LoadDefaultConfig Error: %v", err.Error())
		return stsAPI, err
	}

	stsClient := sts.NewFromConfig(cfg)
	stsAPI.STS = stsClient
	return stsAPI, nil
}

func (api *stsAPI) AssumeRole(roleArn, roleSessionName string, duration int32) (*sts.AssumeRoleOutput, error) {
	assumeRoleInput := sts.AssumeRoleInput{
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String(roleSessionName),
		DurationSeconds: aws.Int32(duration),
	}

	return api.STS.AssumeRole(context.TODO(), &assumeRoleInput)
}
