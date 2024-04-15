package helpers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func SendInternalError(c *gin.Context, err error) {
	sendCustomErrorMessageWithCallerLogs(c, http.StatusInternalServerError, "internal server error", "Error", err)
}

func SendCustomInternalError(c *gin.Context, externalMsg string, err error) {
	sendCustomErrorMessageWithCallerLogs(c, http.StatusInternalServerError, externalMsg, "Error", err)
}

func SendCustomErrorMessage(c *gin.Context, httpCode int, externalMsg string, err error) {
	sendCustomErrorMessageWithCallerLogs(c, httpCode, externalMsg, "Error", err)
}

func SendCustomErrorMessageWithLogs(c *gin.Context, httpCode int, externalMsg, internalMsg string, err error) {
	sendCustomErrorMessageWithCallerLogs(c, httpCode, externalMsg, internalMsg, err)
}

func sendCustomErrorMessageWithCallerLogs(c *gin.Context, httpCode int, externalMsg string,
	internalMsg string, err error) {
	log.Error().Stack().Err(err).Caller(2).Msg(internalMsg)
	c.JSON(httpCode, gin.H{"error": externalMsg})
}
