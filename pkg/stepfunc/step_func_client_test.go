package stepfunc

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/golang/mock/gomock"

	"github.com/pranav-patil/go-serverless-api/pkg/stepfunc/mocks"
	"github.com/stretchr/testify/suite"
)

type DownloadBookmarksInput struct {
	UserId           string `json:"userId"`
	OperationId      string `json:"operationId"`
	DeviceId         string `json:"deviceId"`
	InstanceId       string `json:"instanceId"`
	Enabled          bool   `json:"enabled"`
	BookmarksVersion string `json:"bookmarksVersion"`
	Checksum         string `json:"fileChkSum"`
	S3PresignedURL   string `json:"presignedUrl"`
	RespToken        string `json:"respToken"`
}

type StepFuncClientTestSuite struct {
	suite.Suite

	ctrl             *gomock.Controller
	mockStepFnClient *mocks.MockAWSStepFuncClient
	api              sfnAPI
}

func TestStepFuncClientSuite(t *testing.T) {
	suite.Run(t, new(StepFuncClientTestSuite))
}

func (s *StepFuncClientTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
}

func (s *StepFuncClientTestSuite) SetupTest() {
	s.T().Log("SetupTest Step Function")
	s.mockStepFnClient = mocks.NewMockAWSStepFuncClient(s.ctrl)
	s.api = sfnAPI{Sfn: s.mockStepFnClient}
}

func (s *StepFuncClientTestSuite) TestStartExecution() {
	stateMachineArn := "test_step_function_arn"
	executionName := "test_execu_name"
	initialState := DownloadBookmarksInput{
		UserId:           "120",
		OperationId:      "2",
		DeviceId:         "23423443",
		InstanceId:       "53354",
		Enabled:          true,
		BookmarksVersion: "1.0.9",
		Checksum:         "checksum",
		S3PresignedURL:   "url://s3/signed",
	}

	var startExecutionInput *sfn.StartExecutionInput

	s.mockStepFnClient.EXPECT().StartExecution(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, p *sfn.StartExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error) {
			startExecutionInput = p
			return &sfn.StartExecutionOutput{}, nil
		}).Times(1)

	err := s.api.StartExecution(stateMachineArn, executionName, initialState)

	s.NoError(err)
	s.Equal(&stateMachineArn, startExecutionInput.StateMachineArn)
	s.Equal(&executionName, startExecutionInput.Name)
	s.True(strings.Contains(*startExecutionInput.Input, "\"deviceId\":\"23423443\""))
}

func (s *StepFuncClientTestSuite) TestSendTaskSuccess() {
	taskToken := "test_token"
	message := "test_message"
	taskSuccessIn := &sfn.SendTaskSuccessInput{
		Output:    &message,
		TaskToken: &taskToken,
	}

	ctx := context.TODO()
	s.mockStepFnClient.EXPECT().SendTaskSuccess(ctx, taskSuccessIn).Return(&sfn.SendTaskSuccessOutput{}, nil).Times(1)

	err := s.api.SendTaskSuccess(taskToken, message)

	s.NoError(err)
}

func (s *StepFuncClientTestSuite) TestSendTaskFailure() {
	taskToken := "test_token"
	errMessage := "test_error_message"
	errCause := "test_error_cause"
	taskSuccessIn := &sfn.SendTaskFailureInput{
		TaskToken: &taskToken,
		Error:     &errMessage,
		Cause:     &errCause,
	}

	ctx := context.TODO()
	s.mockStepFnClient.EXPECT().SendTaskFailure(ctx, taskSuccessIn).Return(&sfn.SendTaskFailureOutput{}, nil).Times(1)

	err := s.api.SendTaskFailure(taskToken, errMessage, errCause)

	s.NoError(err)
}
