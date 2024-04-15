package util

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type IPUtilTestSuite struct {
	suite.Suite
}

func TestIPUtilSuite(t *testing.T) {
	suite.Run(t, new(IPUtilTestSuite))
}

func (s *IPUtilTestSuite) TestValidateIPWithIPv4One() {
	testIP := "10.0.0.1"
	actual, err := ValidateIP(testIP)
	s.Equal("10.0.0.1/32", actual.String())
	s.NoError(err)
}

func (s *IPUtilTestSuite) TestValidateIPWithIPv4Two() {
	testIP := "172.28.50.143"
	actual, err := ValidateIP(testIP)
	s.Equal("172.28.50.143/32", actual.String())
	s.NoError(err)
}

func (s *IPUtilTestSuite) TestValidateIPWithIPv4Three() {
	testIP := "192.168.1.200"
	actual, err := ValidateIP(testIP)
	s.Equal("192.168.1.200/32", actual.String())
	s.NoError(err)
}

func (s *IPUtilTestSuite) TestValidateIPWithIPv4Four() {
	testIP := "216.104.20.24"
	actual, err := ValidateIP(testIP)
	s.Equal("216.104.20.24/32", actual.String())
	s.NoError(err)
}

func (s *IPUtilTestSuite) TestValidateIPWithCIDRv4() {
	testCidr := "10.0.0.0/16"
	actual, err := ValidateIP(testCidr)
	s.Equal("10.0.0.0/16", actual.String())
	s.NoError(err)
}

func (s *IPUtilTestSuite) TestValidateIPWithSingleAddrCIDR() {
	testCidr := "198.51.100.0/32"
	actual, err := ValidateIP(testCidr)
	s.Equal("198.51.100.0/32", actual.String())
	s.NoError(err)
}

func (s *IPUtilTestSuite) TestValidateIPWithIPv4OutOfRange() {
	testIP := "256.0.0.0"
	actual, err := ValidateIP(testIP)
	s.Equal("invalid Prefix", actual.String())
	s.Equal("256.0.0.0 is neither a valid IPv4 nor CIDR", err.Error())
}

func (s *IPUtilTestSuite) TestValidateIPWithInvalidOctet() {
	testIP := "192.0.2.00"
	actual, err := ValidateIP(testIP)
	s.Equal("invalid Prefix", actual.String())
	s.Equal("192.0.2.00 is neither a valid IPv4 nor CIDR", err.Error())
}

func (s *IPUtilTestSuite) TestValidateIPWithSpecialCharsInIPv4() {
	testIP := "192.168.*.*"
	actual, err := ValidateIP(testIP)
	s.Equal("invalid Prefix", actual.String())
	s.Equal("192.168.*.* is neither a valid IPv4 nor CIDR", err.Error())
}

func (s *IPUtilTestSuite) TestValidateIPWithAlphaCharsInIPv4() {
	testIP := "192.0.2.db8"
	actual, err := ValidateIP(testIP)
	s.Equal("invalid Prefix", actual.String())
	s.Equal("192.0.2.db8 is neither a valid IPv4 nor CIDR", err.Error())
}

func (s *IPUtilTestSuite) TestValidateIPWithInvalidCIDRv4() {
	testCidr := "10.0.0.0/33"
	actual, err := ValidateIP(testCidr)
	s.Equal("invalid Prefix", actual.String())
	s.Equal("10.0.0.0/33 is neither a valid IPv4 nor CIDR", err.Error())
}

func (s *IPUtilTestSuite) TestValidateIPWithIPv6() {
	testIP := "2001:db8::"
	actual, err := ValidateIP(testIP)
	s.Equal("invalid Prefix", actual.String())
	s.Equal("2001:db8:: is invalid because IPv6 is not supported", err.Error())
}

func (s *IPUtilTestSuite) TestValidateIPWithCIDRv6() {
	testCidr := "2001:db8::/48"
	actual, err := ValidateIP(testCidr)
	s.Equal("invalid Prefix", actual.String())
	s.Equal("2001:db8::/48 is invalid because IPv6 is not supported", err.Error())
}

func (s *IPUtilTestSuite) TestValidateIPWithNotKnownAddr() {
	testCidr := "0.0.0.0"
	actual, err := ValidateIP(testCidr)
	s.Equal("0.0.0.0/32", actual.String())
	s.NoError(err)
}

func (s *IPUtilTestSuite) TestIP4ToInteger() {
	testCases := []struct {
		testName          string
		inputIP           string
		expectedFirstAddr uint32
		expectedLastAddr  uint32
	}{
		{"IP Address",
			"192.168.9.12",
			3232237836,
			3232237836},

		{"CIDR Address",
			"10.0.0.0/24",
			167772160,
			167772415},
		{
			"testAddr: 255.255.255.255",
			"255.255.255.255",
			4294967295,
			4294967295,
		},
		{
			"testAddr: 0.0.0.0",
			"0.0.0.0",
			0,
			0,
		},
		{
			"testAddr: 255.0.0.0/8",
			"255.0.0.0/8",
			4278190080,
			4294967295,
		},
		{
			"testAddr: 128.0.0.0/8",
			"128.0.0.0/8",
			2147483648,
			2164260863,
		},
		{
			"testAddr: 255.255.255.255/32",
			"255.255.255.255/32",
			4294967295,
			4294967295,
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.testName)
			firstAddr, lastAddr, err := GetFirstLastDecimalAddress(testCase.inputIP)
			s.Nil(err)
			s.EqualValues(testCase.expectedFirstAddr, firstAddr)
			s.EqualValues(testCase.expectedLastAddr, lastAddr)
		})
	}
}

func (s *IPUtilTestSuite) TestConvertToIP4() {
	testCases := []struct {
		testName     string
		inputIntIP   uint32
		expectedAddr string
	}{
		{"IP Address",
			3232237836,
			"192.168.9.12"},

		{"CIDR Address",
			167837696,
			"10.1.0.0"},
		{
			"Test Decimal: 4294967295",
			4294967295,
			"255.255.255.255",
		},
		{
			"Test Decimal: 0",
			0,
			"0.0.0.0",
		},
		{
			"Test Decimal: 2147483648",
			2147483648,
			"128.0.0.0",
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.testName)
			ipAddr := ConvertToIP4(testCase.inputIntIP)
			s.EqualValues(testCase.expectedAddr, ipAddr)
		})
	}
}
