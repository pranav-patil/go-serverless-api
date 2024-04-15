package stepfunc

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

//go:generate mockgen -destination mocks/aws_sfn_client_mock.go -package mocks . AWSStepFuncClient

type AWSStepFuncClient interface {
	StartExecution(ctx context.Context, params *sfn.StartExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error)
	SendTaskSuccess(ctx context.Context, params *sfn.SendTaskSuccessInput, optFns ...func(*sfn.Options)) (*sfn.SendTaskSuccessOutput, error)
	SendTaskFailure(ctx context.Context, params *sfn.SendTaskFailureInput, optFns ...func(*sfn.Options)) (*sfn.SendTaskFailureOutput, error)
}
