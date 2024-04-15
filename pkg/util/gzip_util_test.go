package util

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type GZipUtilTestSuite struct {
	suite.Suite
}

func TestGZipUtilSuite(t *testing.T) {
	suite.Run(t, new(GZipUtilTestSuite))
}

func (s *GZipUtilTestSuite) TestGZipCompressAndDecompress() {
	testCases := []struct {
		testName     string
		inputPayload string
	}{
		{"Simple Text",
			"This is a test"},

		{"Payload with spaces, tabs, new lines, quotes etc",
			`{"bookmarks": [
				{
					"url": "https://docs.ai21.com/docs/jurassic-2-models"
				},
				{
					"url": "https://jalammar.github.io/illustrated-transformer/"
				},
				{
					"url": "https://chat.openai.com"
				}
			]}`},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.testName)
			result, err := Compress(testCase.inputPayload)
			s.Nil(err)
			actualValue, err := Decompress(result)
			s.Nil(err)
			s.EqualValues(testCase.inputPayload, actualValue)
		})
	}
}

func (s *GZipUtilTestSuite) TestCreateZipFile() {
	files := map[string]string{"foo.txt": "version:1.4.6.8", "bar.pkg": "4,5,6,7,2,5,54,75"}

	buf, err := CreateZipFile(files)
	s.Nil(err)
	s.NotNil(buf)
}

func (s *GZipUtilTestSuite) TestCreateTarFile() {
	files := map[string]string{"foo.txt": "version:1.4.6.8", "bar.pkg": "4,5,6,7,2,5,54,75"}

	buf, err := CreateTarFile(files)
	s.Nil(err)
	s.NotNil(buf)
}
