package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pranav-patil/go-serverless-api/func/api/middleware"
	"github.com/pranav-patil/go-serverless-api/pkg/launchdarkly"
	"github.com/rs/zerolog/log"
)

func Validate() func(c *gin.Context) {
	return func(c *gin.Context) {
		aid, exists := c.Get(middleware.UserIDCxt)
		if !exists {
			log.Error().Msg("Unable to get UserID from context")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "unable to find JWT user ID"})
		}
		accountId := aid.(string)

		ld, exists := c.Get(middleware.LaunchDarklyCxt)
		if !exists {
			log.Error().Msg("Unable to get LaunchDarkly client from context")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		ldClient := ld.(*launchdarkly.Launchdarkly)

		if !ldClient.IsBookmarkFeatureEnabled(accountId) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "feature is not available"})
		}

		c.Next()
	}
}
