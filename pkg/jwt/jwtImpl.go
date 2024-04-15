package jwt

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/awslabs/aws-lambda-go-api-proxy/core"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/rs/zerolog/log"
)

type Jwt struct {
	account string
	role    EmproviseRole
}

type EmproviseClaims struct {
	jwt.RegisteredClaims
	Emprovise `json:"application"`
}

type Emprovise struct {
	Principal `json:"principal"`
}

type Principal struct {
	Type      string `json:"type"`
	AccountID string `json:"account"`
	Role      string `json:"role"`
	URN       string `json:"urn,omitempty"`
}

type EmproviseRole int

const (
	RoleFullAccess EmproviseRole = iota
	RoleAuditor
	RoleReadOnly
	RoleUnknown
)

var cloudOneRoleMap = map[string]EmproviseRole{
	"full-access": RoleFullAccess,
	"auditor":     RoleAuditor,
	"read-only":   RoleReadOnly,
}

func (role EmproviseRole) String() string {
	for k, v := range cloudOneRoleMap {
		if v == role {
			return k
		}
	}
	return "unknown"
}

func getEmproviseRole(claimRole string) (EmproviseRole, error) {
	// format: urn:emprovise:identity:us-east-1:10:role/full-access

	var roleName string
	if index := strings.LastIndex(claimRole, "/"); index != -1 {
		roleName = claimRole[index+1:]
	} else {
		return RoleUnknown, fmt.Errorf("invalid role string")
	}

	cloudOneRole, ok := cloudOneRoleMap[roleName]
	if !ok {
		cloudOneRole = RoleAuditor
		log.Warn().Msgf("unknown emprovise role '%s', use default role 'auditor'", roleName)
	}

	return cloudOneRole, nil
}

func NewJwt(tokenString string) (*Jwt, error) {
	if env.IsLocalOrTestEnv() {
		accountID := strconv.Itoa(mockJwtMap()[tokenString])
		return &Jwt{accountID, RoleUnknown}, nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &EmproviseClaims{}, func(token *jwt.Token) (interface{}, error) {
		return nil, nil
	})

	// token validation
	if err != nil {
		var ev *jwt.ValidationError

		if errors.As(err, &ev) {
			switch {
			case ev.Errors&jwt.ValidationErrorMalformed != 0:
				return nil, fmt.Errorf("token is malformed")
			case ev.Errors&jwt.ValidationErrorExpired != 0:
				return nil, fmt.Errorf("token is expired")
			}
		} else {
			return nil, fmt.Errorf("unknown error")
		}
	}

	// if valid Emprovise token
	var claims *EmproviseClaims
	var ok bool
	if claims, ok = token.Claims.(*EmproviseClaims); ok {
		if claims.Principal.AccountID == "" {
			return nil, fmt.Errorf("account is empty")
		}

		if claims.Principal.Role == "" {
			return nil, fmt.Errorf("role is empty")
		}
	} else {
		return nil, fmt.Errorf("fail to convert emprovise claims")
	}

	cloudOneRole, err := getEmproviseRole(claims.Principal.Role)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("accountId: %s, role: %s", claims.Principal.AccountID, cloudOneRole)

	return &Jwt{claims.Principal.AccountID, cloudOneRole}, nil
}

func (p *Jwt) Account() string {
	return p.account
}

func (p *Jwt) Role() EmproviseRole {
	return p.role
}

func GetAuthToken(c *gin.Context) (string, error) {
	if env.IsLocalOrTestEnv() {
		jwtHeader := c.Request.Header.Get("Authorization")

		if len(jwtHeader) > 0 {
			return strings.TrimSpace(strings.TrimPrefix(jwtHeader, "apikey")), nil
		}
	}

	apiGWReqContext, _ := core.GetAPIGatewayContextFromContext(c.Request.Context())

	auth, ok := apiGWReqContext.Authorizer["Authorization"]
	if !ok {
		return "", fmt.Errorf("'Authorization' field not found")
	}

	authStr, ok := auth.(string)
	if !ok {
		return "", fmt.Errorf("fail to convert 'Authorization' field to string")
	}

	return strings.TrimPrefix(authStr, "Bearer "), nil
}

func mockJwtMap() map[string]int {
	return map[string]int{
		"AAA": 0,
		"BBB": 1,
		"CCC": 2,
		"DDD": 3,
		"EEE": 4,
		"24EE6E9B-E2A2-5E0F-E351-A1CC035D444F:rZixVA90C70BkWOxzbBb06cwEcN4ITwJClRtpv+9yAQ=": 5,
		"FFF": 6,
		"GGG": 7,
	}
}
