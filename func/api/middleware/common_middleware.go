package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pranav-patil/go-serverless-api/pkg/jwt"
	"github.com/pranav-patil/go-serverless-api/pkg/launchdarkly"
	"github.com/rs/zerolog/log"
)

const (
	UserIDCxt       string = "USER_ID"
	LaunchDarklyCxt string = "LD_CLIENT"
	JWTToken        string = "JWT_TOKEN"
)

func Attach(router *gin.Engine) {
	router.Use(extractUserIDFromJWT())
	router.Use(attachLDClient())
}

func attachLDClient() gin.HandlerFunc {
	ldClient, err := launchdarkly.NewLaunchDarklyClient()

	return func(c *gin.Context) {
		if err != nil {
			log.Error().Msgf("Fail to initialize LaunchDarkly client: %s", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}

		c.Set(LaunchDarklyCxt, ldClient)
	}
}

func extractUserIDFromJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken, err := jwt.GetAuthToken(c)
		if err != nil {
			log.Error().Msgf("Error in extracting JWTToken from APIGatewayProxyRequestContext: %v", err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "authorization token empty"})
		}

		jwtToken, err := jwt.NewJwt(authToken)
		if err != nil {
			log.Error().Msgf("Error in fetching JWTToken: %v", err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "error in fetching JWT account id"})
		}

		log.Debug().Msgf("Setting Context with User Id: %s", jwtToken.Account())

		c.Set(JWTToken, authToken)
		c.Set(UserIDCxt, jwtToken.Account())
	}
}
