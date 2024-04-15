package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pranav-patil/go-serverless-api/func/api/helpers"
	"github.com/pranav-patil/go-serverless-api/func/api/middleware"
	"github.com/pranav-patil/go-serverless-api/func/api/models"
	"github.com/pranav-patil/go-serverless-api/pkg/s3"
)

func PostBookmarks(context *gin.Context) {
	var err error
	bookmarkList := models.BookmarkList{}

	if err = context.BindJSON(&bookmarkList); err != nil {
		helpers.SendCustomErrorMessage(context, http.StatusBadRequest, "invalid json payload", err)
		return
	}
	validBookmarks, rejectedBookmarks := helpers.ValidateBookmarks(bookmarkList.BookmarkEntry)

	if len(rejectedBookmarks) > 0 {
		context.JSON(http.StatusBadRequest, &models.InValidBookmarksResponse{
			Error:        "Invalid Bookmarks",
			BookmarkList: rejectedBookmarks},
		)
		return
	}

	userId := context.GetString(middleware.UserIDCxt)

	dynamodbClient, err := NewDynamoDBClient()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

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
	distEntryPath := helpers.GetUserBookmarksS3Path(distribution)

	if distEntryPath != "" {
		bookmarkList, err = getExistingBookmarks(s3Client, validBookmarks, bucketName, distEntryPath)
		if err != nil {
			helpers.SendInternalError(context, err)
			return
		}
	} else {
		bookmarkList.BookmarkEntry = validBookmarks
	}

	content, err := json.Marshal(&bookmarkList)
	if err != nil {
		helpers.SendCustomErrorMessage(context, http.StatusBadRequest, "invalid json payload", err)
		return
	}

	err = helpers.AddBookmarksInS3Bucket(dynamodbClient, s3Client, distribution, userId, bucketName, JSON, s3.GZip, &content)
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	context.JSON(http.StatusCreated, &models.BookmarksResponse{
		BookmarkList: bookmarkList.BookmarkEntry,
		TotalCount:   len(bookmarkList.BookmarkEntry),
	})
}

func DeleteBookmarks(context *gin.Context) {
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

	previousVersion := helpers.GetUserBookmarksS3Path(distribution)
	if previousVersion == "" {
		context.JSON(http.StatusForbidden, gin.H{"error": "no bookmarks found to delete"})
		return
	}

	s3Client, err := NewS3Client()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	err = helpers.AddOrUpdateUserBookmarks(dynamodbClient, distribution, userId,
		fmt.Sprint(distribution.LatestVersion, helpers.DeletedVersionSuffix), true)

	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	bucketName := os.Getenv("BOOKMARKS_BUCKET")
	err = s3Client.DeleteObject(bucketName, previousVersion)
	if err != nil {
		helpers.SendCustomInternalError(context, fmt.Sprintf("failure in deleting all Bookmarks for userId %s", userId), err)
		return
	}

	context.JSON(http.StatusAccepted, gin.H{"message": "Bookmarks deletion complete"})
}

func getExistingBookmarks(s3Client s3.S3Client, newBookmarks []models.BookmarkEntry, bucketName, path string) (models.BookmarkList, error) {
	bookmarkList := models.BookmarkList{}

	data, err := s3Client.GetObject(bucketName, path)
	if err != nil {
		return bookmarkList, err
	}

	if err := json.Unmarshal(data, &bookmarkList); err != nil {
		return bookmarkList, err
	}

	bookmarkList.BookmarkEntry = removeDuplicates(append(bookmarkList.BookmarkEntry, newBookmarks...))
	return bookmarkList, nil
}

func removeDuplicates(slices []models.BookmarkEntry) []models.BookmarkEntry {
	allKeys := make(map[string]bool)
	list := []models.BookmarkEntry{}
	for _, item := range slices {
		if _, value := allKeys[item.URL]; !value {
			allKeys[item.URL] = true
			list = append(list, item)
		}
	}
	return list
}
