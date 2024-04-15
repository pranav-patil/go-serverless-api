package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pranav-patil/go-serverless-api/func/api/helpers"
	"github.com/pranav-patil/go-serverless-api/func/api/middleware"
	"github.com/pranav-patil/go-serverless-api/func/api/models"
	"github.com/pranav-patil/go-serverless-api/pkg/s3"
)

const (
	JSON             string = "application/json"
	CSV              string = "text/csv"
	defaultPageLimit        = 100
)

var (
	NewS3Client = s3.NewS3Client
)

func GetBookmarks(context *gin.Context) {
	bookmarks := models.BookmarkList{}

	pageLimit, err := strconv.Atoi(context.Query("limit"))
	if err != nil {
		pageLimit = defaultPageLimit
	}

	var lastEvalRecord string

	cursor, err := url.PathUnescape(context.Query("cursor"))
	if err == nil {
		var decodedKey []byte
		decodedKey, err = base64.StdEncoding.DecodeString(cursor)

		if err == nil && len(decodedKey) > 0 {
			lastEvalRecord = string(decodedKey)
		}
	}

	s3Client, err := NewS3Client()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	dynamodbClient, err := NewDynamoDBClient()
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

	if err = json.Unmarshal(data, &bookmarks); err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	bookmarks.BookmarkEntry, err = paginateBookmarks(bookmarks.BookmarkEntry, lastEvalRecord, pageLimit)
	if err != nil {
		helpers.SendCustomErrorMessage(context, http.StatusNotFound, "Bookmarks page not found", err)
		return
	}

	var nextToken string
	if bkmLength := len(bookmarks.BookmarkEntry); len(bookmarks.BookmarkEntry) == pageLimit {
		value := bookmarks.BookmarkEntry[bkmLength-1].URL
		nextToken = base64.StdEncoding.EncodeToString([]byte(value))
		nextToken = url.PathEscape(nextToken)
	}

	response := models.BookmarksResponse{
		TotalCount:   len(bookmarks.BookmarkEntry),
		Next:         nextToken,
		BookmarkList: bookmarks.BookmarkEntry}

	context.JSON(http.StatusOK, &response)
}

func PutBookmarks(context *gin.Context) {
	bookmarks := models.BookmarkList{}
	var content []byte
	var err error
	var validBookmarks, rejectedBookmarks []models.BookmarkEntry
	contentType := context.Request.Header.Get("Content-Type")

	if strings.EqualFold(contentType, JSON) {
		if err = context.BindJSON(&bookmarks); err != nil {
			helpers.SendCustomErrorMessage(context, http.StatusBadRequest, "invalid json payload", err)
			return
		}
		validBookmarks, rejectedBookmarks = helpers.ValidateBookmarks(bookmarks.BookmarkEntry)
	} else if strings.EqualFold(contentType, CSV) {
		content, err = context.GetRawData()
		if err != nil {
			helpers.SendCustomErrorMessage(context, http.StatusBadRequest, "invalid csv payload", err)
			return
		}

		validBookmarks, rejectedBookmarks, err = helpers.ConvertCSVAndValidateBookmarks(string(content))
		if err != nil {
			helpers.SendCustomErrorMessage(context, http.StatusBadRequest, "invalid csv payload", err)
			return
		}
	} else {
		context.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("invalid content type %v", contentType)})
		return
	}

	if len(rejectedBookmarks) > 0 {
		context.JSON(http.StatusBadRequest, &models.InValidBookmarksResponse{
			Error:        "Invalid Bookmarks",
			BookmarkList: rejectedBookmarks},
		)
		return
	}

	bookmarks.BookmarkEntry = validBookmarks
	content, err = json.Marshal(&bookmarks)
	if err != nil {
		helpers.SendCustomErrorMessage(context, http.StatusBadRequest, "invalid json payload", err)
		return
	}

	dynamodbClient, err := NewDynamoDBClient()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	userId := context.GetString(middleware.UserIDCxt)
	distribution := helpers.GetBookmarkByUser(dynamodbClient, userId)
	if helpers.IsDistributionPending(distribution) {
		context.JSON(http.StatusForbidden, gin.H{"error": "distribution is in Progress"})
		return
	}

	s3Client, err := NewS3Client()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	bucketName := os.Getenv("BOOKMARKS_BUCKET")
	err = helpers.AddBookmarksInS3Bucket(dynamodbClient, s3Client, distribution, userId, bucketName, JSON, s3.GZip, &content)
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	context.JSON(http.StatusCreated, &models.BookmarksResponse{
		BookmarkList: bookmarks.BookmarkEntry,
		TotalCount:   len(bookmarks.BookmarkEntry),
	})
}

func paginateBookmarks(bookmarks []models.BookmarkEntry, lastEntry string, limit int) ([]models.BookmarkEntry, error) {
	index := -1

	if lastEntry != "" {
		for i, entry := range bookmarks {
			if index < 0 && entry.URL == lastEntry {
				index = i + 1
				break
			}
		}
	}

	if index == -1 {
		index = 0
	}

	if (len(bookmarks) - 1) > (index + limit) {
		return bookmarks[index:(index + limit)], nil
	} else if len(bookmarks) == index {
		return nil, fmt.Errorf("no records found from %s", lastEntry)
	} else {
		return bookmarks[index:], nil
	}
}
