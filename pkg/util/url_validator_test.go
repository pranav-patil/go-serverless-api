package util

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type URLValidatorTestSuite struct {
	suite.Suite
}

func TestURLValidatorSuite(t *testing.T) {
	suite.Run(t, new(URLValidatorTestSuite))
}

func (s *URLValidatorTestSuite) TestValidURLs() {
	testCases := []struct {
		testName      string
		inputURL      string
		expectedError bool
	}{
		{"HTTP Valid URL",
			"https://example.com",
			false},

		{"Invalid HTPS URL",
			"htps://www.run.ai/guides/gpu-deep-learning/best-gpu-for-deep-learning",
			true},
		{"Invalid URL without colon",
			"https/www.simplilearn.com/keras-vs-tensorflow-vs-pytorch-article",
			true},
		{
			"FTP Valid URL",
			"ftp://example.com",
			false,
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.testName)
			result := ValidateURL(testCase.inputURL)
			if testCase.expectedError {
				s.Error(result)
			} else {
				s.NoError(result)
			}
		})
	}
}
