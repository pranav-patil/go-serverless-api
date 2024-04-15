package util

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ConverterUtilTestSuite struct {
	suite.Suite
}

func TestConverterUtilSuite(t *testing.T) {
	suite.Run(t, new(ConverterUtilTestSuite))
}

type Job struct {
	JobId          int64     `dynamodbav:"jobId" partitionKey:"JID"`
	Version        string    `dynamodbav:"version" sortKey:"VID"`
	Type           string    `dynamodbav:"type,omitempty"`
	StartTimestamp time.Time `dynamodbav:"startTs"`
}

func (s *ConverterUtilTestSuite) TestStructToMap() {
	job := Job{
		JobId:          23424,
		Version:        "1.0",
		Type:           "Main",
		StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
	}
	jobMap, _ := StructToMap(&job, "dynamodbav", false)

	expectedMap := map[string]interface{}{
		"jobId":   int64(23424),
		"version": "1.0",
		"type":    "Main",
		"startTs": time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
	}

	eq := reflect.DeepEqual(jobMap, expectedMap)

	s.Equal(true, eq)
}

func (s *ConverterUtilTestSuite) TestStructToMapWithEmptyType() {
	job := Job{
		JobId:          23424,
		Version:        "1.0",
		Type:           "",
		StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
	}
	jobMap, _ := StructToMap(&job, "dynamodbav", false)

	expectedMap := map[string]interface{}{
		"jobId":   int64(23424),
		"version": "1.0",
		"startTs": time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
	}

	s.Equal(3, len(jobMap))

	eq := reflect.DeepEqual(jobMap, expectedMap)

	s.Equal(true, eq)
}

func (s *ConverterUtilTestSuite) TestStructToMapFailsWhenNumberFieldsNotInt64() {
	currentUTCTime := time.Now().UTC()

	job := Job{
		JobId:          23424,
		Version:        "1.0",
		Type:           "Main",
		StartTimestamp: currentUTCTime,
	}
	jobMap, _ := StructToMap(&job, "dynamodbav", false)

	expectedMap := map[string]interface{}{
		"jobId":   23424,
		"version": "1.0",
		"type":    "Main",
		"startTs": time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
	}

	eq := reflect.DeepEqual(jobMap, expectedMap)

	s.Equal(false, eq)
}

func (s *ConverterUtilTestSuite) TestStructToMapWithTags() {
	testCases := []struct {
		testName       string
		inputTags      []string
		inputStruct    *Job
		ignoreDefaults bool
		outputMap      map[string]interface{}
	}{
		{"Remove fields with partitionKey and sortKey",
			[]string{"partitionKey", "sortKey"},
			&Job{
				JobId:          23424,
				Version:        "1.0",
				Type:           "Main",
				StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			},
			false,
			map[string]interface{}{
				"type":    "Main",
				"startTs": time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			}},

		{"Remove fields with partitionKey",
			[]string{"partitionKey"},
			&Job{
				JobId:          23424,
				Version:        "1.0",
				Type:           "Main",
				StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			},
			false,
			map[string]interface{}{
				"version": "1.0",
				"type":    "Main",
				"startTs": time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			}},

		{"Remove fields with default values when ignore defaults true",
			[]string{},
			&Job{
				JobId:          0,
				Version:        "1.0",
				Type:           "",
				StartTimestamp: time.Time{},
			},
			true,
			map[string]interface{}{
				"version": "1.0",
			}},

		{"Remove fields with default values when ignore defaults false",
			[]string{},
			&Job{
				JobId:          0,
				Version:        "1.0",
				Type:           "",
				StartTimestamp: time.Time{},
			},
			false,
			map[string]interface{}{
				"jobId":   int64(0),
				"version": "1.0",
				"startTs": time.Time{},
			}},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.inputStruct)
			actualMap, err := StructToMap(testCase.inputStruct, "dynamodbav", testCase.ignoreDefaults,
				testCase.inputTags...)

			s.Nil(err)
			s.EqualValues(testCase.outputMap, actualMap)
		})
	}
}

func (s *ConverterUtilTestSuite) TestGetStructField() {
	testCases := []struct {
		testName       string
		structInstance *Job
		fieldName      string
		outputValue    interface{}
	}{
		{"String field",
			&Job{
				JobId:          23424,
				Version:        "1.0",
				Type:           "Main",
				StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			},
			"Version",
			"1.0"},

		{"Integer field",
			&Job{
				JobId:          23424,
				Version:        "1.0",
				Type:           "Main",
				StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			},
			"JobId",
			23424},

		{"Timestamp field",
			&Job{
				JobId:          23424,
				Version:        "1.0",
				Type:           "Main",
				StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			},
			"StartTimestamp",
			time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.testName)
			actualValue := GetStructField(testCase.structInstance, testCase.fieldName)
			s.EqualValues(testCase.outputValue, actualValue)
		})
	}
}

func (s *ConverterUtilTestSuite) TestSetStructField() {
	testCases := []struct {
		testName       string
		structInstance *Job
		fieldName      string
		inputValue     interface{}
	}{
		{"String field",
			&Job{
				JobId:          23424,
				Version:        "1.0",
				Type:           "Main",
				StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			},
			"Version",
			"56.9"},

		{"Integer field",
			&Job{
				JobId:          23424,
				Version:        "1.0",
				Type:           "Main",
				StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			},
			"JobId",
			int64(5675567)},

		{"Timestamp field",
			&Job{
				JobId:          23424,
				Version:        "1.0",
				Type:           "Main",
				StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			},
			"StartTimestamp",
			time.Date(2019, 04, 17, 20, 34, 58, 651387237, time.UTC)},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.testName)
			SetStructField(testCase.structInstance, testCase.fieldName, testCase.inputValue)
			actualValue := GetStructField(testCase.structInstance, testCase.fieldName)
			s.EqualValues(testCase.inputValue, actualValue)
		})
	}
}

func (s *ConverterUtilTestSuite) TestContains() {
	strArray := []string{"James", "Wagner", "Christene", "Mike"}

	s.Equal(false, Contains(strArray, "Jack"))
	s.Equal(true, Contains(strArray, "James"))
}
