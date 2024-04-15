package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	h "github.com/pranav-patil/go-serverless-api/func/api/handler"
)

func APIRouter(router *gin.Engine) {
	apiRouter := router.Group("/emprovise/api").Use(h.Validate())

	apiRouter.GET("/bookmarks", h.GetBookmarks)
	apiRouter.POST("/bookmarks", h.PostBookmarks)
	apiRouter.PUT("/bookmarks", h.PutBookmarks)
	apiRouter.DELETE("/bookmarks", h.DeleteBookmarks)

	apiRouter.HEAD("/bookmarks/:url", h.FindBookmarkEntry)
	apiRouter.DELETE("/bookmarks/:url", h.FindAndDeleteBookmarkEntry)

	apiRouter.POST("/bookmarks/summary", h.DistributeBookmarks)
	apiRouter.GET("/bookmarks/pages", h.GetDistributedBookmarks)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Service not found"})
	})
}
