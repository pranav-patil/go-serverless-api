package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pranav-patil/go-serverless-api/func/api/helpers"
	"github.com/pranav-patil/go-serverless-api/func/api/middleware"
	"github.com/pranav-patil/go-serverless-api/func/api/models"
	"github.com/pranav-patil/go-serverless-api/pkg/s3"
	"github.com/pranav-patil/go-serverless-api/pkg/util"
)

func FindBookmarkEntry(context *gin.Context) {
	searchURL, err := url.PathUnescape(context.Param("url"))
	if err != nil {
		helpers.SendCustomErrorMessage(context, http.StatusBadRequest, err.Error(), err)
		return
	}
	err = util.ValidateURL(searchURL)
	if err != nil {
		helpers.SendCustomErrorMessage(context, http.StatusBadRequest, err.Error(), err)
		return
	}

	dynamodbClient, err := NewDynamoDBClient()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	s3Client, err := NewS3Client()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	userId := context.GetString(middleware.UserIDCxt)
	data, err := helpers.GetBookmarksS3Object(dynamodbClient, s3Client, userId)
	if err != nil {
		helpers.SendCustomErrorMessage(context, http.StatusNotFound, "Bookmarks not found", err)
		return
	}

	bookmarkList := models.BookmarkList{}
	if err = json.Unmarshal(data, &bookmarkList); err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	found := false

	for _, entry := range bookmarkList.BookmarkEntry {
		if strings.Compare(searchURL, entry.URL) == 0 {
			found = true
			break
		}
	}

	if found {
		context.Status(http.StatusNoContent)
	} else {
		context.JSON(http.StatusNotFound, gin.H{"error": "no entry found for the input bookmark url"})
	}
}

func FindAndDeleteBookmarkEntry(context *gin.Context) {
	url, err := url.PathUnescape(context.Param("url"))
	if err != nil {
		helpers.SendCustomErrorMessage(context, http.StatusBadRequest, err.Error(), err)
		return
	}
	if err = util.ValidateURL(url); err != nil {
		helpers.SendCustomErrorMessage(context, http.StatusBadRequest, err.Error(), err)
		return
	}

	dynamodbClient, err := NewDynamoDBClient()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	userId := context.GetString(middleware.UserIDCxt)
	distribution := helpers.GetBookmarkByUser(dynamodbClient, userId)
	if distribution == nil {
		err = fmt.Errorf("distribution entry not found for userId %s", userId)
		helpers.SendCustomErrorMessage(context, http.StatusNotFound, "Bookmarks not found", err)
		return
	}

	bucketName := os.Getenv("BOOKMARKS_BUCKET")
	distEntryPath := helpers.GetUserBookmarksS3Path(distribution)
	if distEntryPath == "" {
		err = fmt.Errorf("bookmarks not found for userId %s", userId)
		helpers.SendCustomErrorMessage(context, http.StatusNotFound, "Bookmarks not found", err)
		return
	}

	s3Client, err := NewS3Client()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	data, err := s3Client.GetObject(bucketName, distEntryPath)
	if err != nil {
		helpers.SendCustomErrorMessage(context, http.StatusNotFound, "Bookmarks not found", err)
		return
	}

	bookmarkList := models.BookmarkList{}
	if err = json.Unmarshal(data, &bookmarkList); err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	deleteMatchingBookmarks(url, &bookmarkList)

	s3Content, err := json.Marshal(bookmarkList)
	if err != nil {
		helpers.SendInternalError(context, err)
	}

	err = helpers.AddBookmarksInS3Bucket(dynamodbClient, s3Client, distribution, userId, bucketName, JSON, s3.GZip, &s3Content)
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	context.Status(http.StatusNoContent)
}

func deleteMatchingBookmarks(searchURL string, bookmarkList *models.BookmarkList) {
	remainingBookmarks := make([]models.BookmarkEntry, len(bookmarkList.BookmarkEntry))
	index := 0

	for _, entry := range bookmarkList.BookmarkEntry {
		if strings.Compare(searchURL, entry.URL) != 0 {
			remainingBookmarks[index] = entry
			index++
		}
	}

	bookmarkList.BookmarkEntry = remainingBookmarks[:index]
}
