package jwt

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/stretchr/testify/suite"
)

const (
	mockAccountID string = "mock-account-id"
)

type JWTTestSuite struct {
	suite.Suite

	ctrl    *gomock.Controller
	context *gin.Context
}

func TestJWTSuite(t *testing.T) {
	suite.Run(t, new(JWTTestSuite))
}

func (s *JWTTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
}

func (s *JWTTestSuite) SetupTest() {
	recorder := httptest.NewRecorder()
	s.context = mockutil.MockGinContext(recorder)
	s.T().Setenv("STAGE", "Production")
}

func (s *JWTTestSuite) TestNewJwt() {
	mockToken := EmproviseJwtGenerator(
		JWTTestIssuer, false, mockAccountID, fmt.Sprintf(JWTTestRoleFmt, RoleFullAccess), JWTPrivateKey)

	jwtInstance, err := NewJwt(mockToken)

	s.NotNil(jwtInstance)
	s.Nil(err)

	s.Equal(mockAccountID, jwtInstance.Account())
	s.Equal(RoleFullAccess, jwtInstance.Role())

	mockToken = EmproviseJwtGenerator(
		JWTTestIssuer, false, mockAccountID, fmt.Sprintf(JWTTestRoleFmt, RoleReadOnly), JWTPrivateKey)
	jwtInstance, err = NewJwt(mockToken)

	s.NotNil(jwtInstance)
	s.Nil(err)

	s.Equal(mockAccountID, jwtInstance.Account())
	s.Equal(RoleReadOnly, jwtInstance.Role())
}

func (s *JWTTestSuite) TestNewJwtMalformed() {
	jwtInstance, err := NewJwt("mock-jwt")

	s.Nil(jwtInstance)
	s.Equal(fmt.Errorf("token is malformed"), err)
}

func (s *JWTTestSuite) TestNewJwtExpired() {
	mockToken := EmproviseJwtGenerator(
		JWTTestIssuer, true, mockAccountID, fmt.Sprintf(JWTTestRoleFmt, RoleFullAccess), JWTPrivateKey)
	jwtInstance, err := NewJwt(mockToken)

	s.Nil(jwtInstance)
	s.Equal(fmt.Errorf("token is expired"), err)
}

func (s *JWTTestSuite) TestNewJwtEmptyAccountID() {
	mockToken := EmproviseJwtGenerator(
		JWTTestIssuer, false, "", fmt.Sprintf(JWTTestRoleFmt, RoleFullAccess), JWTPrivateKey)
	jwtInstance, err := NewJwt(mockToken)

	s.Nil(jwtInstance)
	s.Equal(fmt.Errorf("account is empty"), err)
}

func (s *JWTTestSuite) TestNewJwtEmptyRole() {
	mockToken := EmproviseJwtGenerator(
		JWTTestIssuer, false, mockAccountID, "", JWTPrivateKey)
	jwtInstance, err := NewJwt(mockToken)

	s.Nil(jwtInstance)
	s.Equal(fmt.Errorf("role is empty"), err)
}

func (s *JWTTestSuite) TestGetEmproviseRole() {
	for _, v := range cloudOneRoleMap {
		role, err := getEmproviseRole(fmt.Sprintf(JWTTestRoleFmt, v))

		s.Nil(err)
		s.Equal(v, role)
	}
}

func (s *JWTTestSuite) TestGetEmproviseRoleInvalidFormat() {
	role, err := getEmproviseRole("mock-invalid-role")

	s.Equal(RoleUnknown, role)
	s.Equal(fmt.Errorf("invalid role string"), err)
}

func (s *JWTTestSuite) TestGetEmproviseRoleInvalidRole() {
	role, err := getEmproviseRole("/xx")

	s.Equal(RoleAuditor, role)
	s.Nil(err)

	role, err = getEmproviseRole(fmt.Sprintf(JWTTestRoleFmt, "mock-role"))

	s.Equal(RoleAuditor, role)
	s.Nil(err)
}

type ctxKey struct{}

func (s *JWTTestSuite) TestGetAuthToken() {
	token := fmt.Sprint("Bearer ", "somedummytoken")

	apiGwyReqCxt := events.APIGatewayProxyRequestContext{
		RequestID:  "x",
		Stage:      "prod",
		Authorizer: map[string]interface{}{"Authorization": token},
	}

	mockContext := s.context.Request.Context()
	mockContext = context.WithValue(mockContext, ctxKey{}, apiGwyReqCxt)

	s.context.Request.Header.Set("Authorization", token)

	_, err := GetAuthToken(s.context)

	s.EqualValues(apiGwyReqCxt, mockContext.Value(ctxKey{}))
	s.NotNil(err)
	s.EqualValues("'Authorization' field not found", err.Error())
}

func (s *JWTTestSuite) TestGetAuthTokenForLocal() {
	s.T().Setenv("STAGE", "")
	s.context.Request.Header.Set("Authorization", "apikey AAA")

	authToken, err := GetAuthToken(s.context)

	s.Nil(err)
	s.EqualValues("AAA", authToken)
}

func (s *JWTTestSuite) TestGetAuthTokenForTesting() {
	s.T().Setenv("STAGE", "testing")
	s.context.Request.Header.Set("Authorization", "apikey BBB")

	authToken, err := GetAuthToken(s.context)

	s.Nil(err)
	s.EqualValues("BBB", authToken)
}

func (s *JWTTestSuite) TestNewJwtNonStaging() {
	s.T().Setenv("STAGE", "")

	jwtInstance, err := NewJwt("BBB")

	s.NotNil(jwtInstance)
	s.Nil(err)

	s.Equal("1", jwtInstance.Account())
	s.Equal(RoleUnknown, jwtInstance.Role())
}
