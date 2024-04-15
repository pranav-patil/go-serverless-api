package sns

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/golang/mock/gomock"

	"github.com/pranav-patil/go-serverless-api/pkg/sns/mocks"
	"github.com/stretchr/testify/suite"
)

var (
	testSNSTopic = "test-sns-topic"
)

type SNSClientTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func TestSNSClientSuite(t *testing.T) {
	suite.Run(t, new(SNSClientTestSuite))
}

func (s *SNSClientTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
}

func (s *SNSClientTestSuite) SetupTest() {
	s.T().Log("SetupTest..")
}

func (s *SNSClientTestSuite) TestPublish() {
	mockSNSClient := mocks.NewMockAWSSNSClient(s.ctrl)
	message := "This is a test sns message"
	messageId := "TEST_MESSAGE_ID"

	ctx := context.TODO()
	input := sns.PublishInput{TopicArn: &testSNSTopic, Message: &message, MessageAttributes: map[string]types.MessageAttributeValue{}}

	mockSNSClient.EXPECT().Publish(ctx, &input).Return(&sns.PublishOutput{MessageId: &messageId}, nil).Times(1)

	api := snsAPI{SNS: mockSNSClient}
	err := api.Publish(testSNSTopic, message)
	if err != nil {
		s.T().Errorf("Expected no error, but got: '%v'", err)
	}

	s.NoError(err)
}
