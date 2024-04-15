package helpers

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pranav-patil/go-serverless-api/func/api/models"
	"github.com/pranav-patil/go-serverless-api/pkg/constant"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb/model"
	"github.com/pranav-patil/go-serverless-api/pkg/s3"
	"github.com/pranav-patil/go-serverless-api/pkg/util"
	"github.com/rs/zerolog/log"
)

const (
	totalVersionDigits         = 4
	DefaultBookmarksVersion    = "1.0.0"
	DeletedVersionSuffix       = "_DELETED"
	DelayedStatusTimeInMinutes = 10
)

var (
	TimeNow = time.Now
)

func GetUserBookmarksS3Path(userBookmarks *model.UserBookmarks) string {
	if userBookmarks == nil || strings.HasSuffix(userBookmarks.LatestVersion, DeletedVersionSuffix) {
		return ""
	}

	return fmt.Sprintf("Bookmarks/%s/%s", userBookmarks.UserId, userBookmarks.LatestVersion)
}

func GetBookmarkByUser(dynamodbClient dynamodb.DynamoDBClient, userId string) *model.UserBookmarks {
	var userBookmarks *model.UserBookmarks
	result, err := dynamodbClient.GetRecordByKey(&model.UserBookmarks{UserId: userId})

	if err != nil {
		log.Warn().Msgf("Failure to fetch user bookmark entry for userId %s: %v", userId, err.Error())
		return nil
	}

	if result != nil {
		userBookmarks = result.(*model.UserBookmarks)
		updateDelayedDistributionToTimeout(userBookmarks)
	}
	return userBookmarks
}

func IsDistributionPending(userBookmarks *model.UserBookmarks) bool {
	return userBookmarks != nil && (userBookmarks.Status == constant.Pending || userBookmarks.Status == constant.BookmarksLocked)
}

func updateDelayedDistributionToTimeout(userBookmarks *model.UserBookmarks) {
	if userBookmarks != nil && userBookmarks.Status == constant.Pending {
		currentTime := TimeNow()
		difference := (currentTime.Sub(userBookmarks.StartTimestamp)).Minutes()

		if difference > DelayedStatusTimeInMinutes {
			userBookmarks.Status = constant.Timeout
			userBookmarks.EndTimestamp = currentTime
			log.Debug().Msgf("Set Bookmark Distribution status for userId %s to timeout after %v mins of pending status.",
				userBookmarks.UserId, difference)
		}
	}
}

func AddOrUpdateUserBookmarks(dynamodbClient dynamodb.DynamoDBClient, userBookmarks *model.UserBookmarks,
	userId, distVersion string, modifiedBookmarks bool) (err error) {
	if userBookmarks == nil {
		userBookmarks = &model.UserBookmarks{
			UserId:            userId,
			SyncEnabled:       true,
			LatestVersion:     distVersion,
			ModifiedBookmarks: modifiedBookmarks,
		}
		err = dynamodbClient.AddRecord(userBookmarks)
	} else {
		userBookmarks.ModifiedBookmarks = modifiedBookmarks
		userBookmarks.LatestVersion = distVersion
		err = dynamodbClient.UpdateRecordsByKey(userBookmarks)
	}
	return err
}

func ConvertCSVAndValidateBookmarks(content string) (validBookmarks, rejectedBookmarks []models.BookmarkEntry, err error) {
	validBookmarks, rejectedBookmarks = []models.BookmarkEntry{}, []models.BookmarkEntry{}
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1
	var csvData [][]string

	if csvData, err = reader.ReadAll(); err != nil {
		log.Error().Msgf("Failure in reading CSV payload: %v", err.Error())
		return nil, nil, err
	}

	for _, each := range csvData {
		entry := models.BookmarkEntry{}
		entry.URL = strings.TrimSpace(each[0])

		if err := util.ValidateURL(entry.URL); err != nil {
			rejectedBookmarks = append(rejectedBookmarks, entry)
		} else {
			validBookmarks = append(validBookmarks, entry)
		}
	}

	return validBookmarks, rejectedBookmarks, nil
}

func ValidateBookmarks(bookmarks []models.BookmarkEntry) (validBookmarks, rejectedBookmarks []models.BookmarkEntry) {
	validBookmarks, rejectedBookmarks = []models.BookmarkEntry{}, []models.BookmarkEntry{}

	for _, entry := range bookmarks {
		if err := util.ValidateURL(entry.URL); err != nil {
			rejectedBookmarks = append(rejectedBookmarks, entry)
		} else {
			validBookmarks = append(validBookmarks, entry)
		}
	}

	return validBookmarks, rejectedBookmarks
}

func GetBookmarksS3Object(dynamodbClient dynamodb.DynamoDBClient, s3Client s3.S3Client, userId string) ([]byte, error) {
	userBookmarks := GetBookmarkByUser(dynamodbClient, userId)
	s3Path := GetUserBookmarksS3Path(userBookmarks)
	if s3Path == "" {
		return nil, fmt.Errorf("no bookmarks exists for userId %s", userId)
	}

	bucketName := os.Getenv("BOOKMARKS_BUCKET")
	return s3Client.GetObject(bucketName, s3Path)
}

func GetIncrementedVersion(userBookmarks *model.UserBookmarks) (string, error) {
	latestVersion := DefaultBookmarksVersion

	if userBookmarks != nil {
		latestVersion = userBookmarks.LatestVersion

		if idx := strings.LastIndex(latestVersion, DeletedVersionSuffix); idx >= 0 {
			latestVersion = latestVersion[:idx]
		}
	}

	var index = strings.LastIndex(latestVersion, ".")

	if index < 0 || index+1 >= len(latestVersion) {
		return "", fmt.Errorf("unable to increment user bookmarks version %s", latestVersion)
	}

	version := latestVersion[index+1:]
	minor, err := strconv.Atoi(version)
	if err != nil {
		return "", err
	}

	return fmt.Sprint(latestVersion[:index+1], minor+1), nil
}

func AddBookmarksInS3Bucket(dynamodbClient dynamodb.DynamoDBClient, s3Client s3.S3Client, userBookmarks *model.UserBookmarks,
	userId, bucket, contentType, encoding string, content *[]byte) error {
	previousVersion := GetUserBookmarksS3Path(userBookmarks)
	distVersion, err := GetIncrementedVersion(userBookmarks)
	if err != nil {
		return err
	}

	distEntryPath := fmt.Sprintf("Bookmarks/%s/%s", userId, distVersion)
	err = s3Client.PutObject(bucket, distEntryPath, contentType, encoding, content)
	if err != nil {
		return err
	}

	err = AddOrUpdateUserBookmarks(dynamodbClient, userBookmarks, userId, distVersion, true)
	if err != nil {
		return err
	}

	if previousVersion != "" {
		err = s3Client.DeleteObject(bucket, previousVersion)
		if err != nil {
			return err
		}
	}

	return nil
}
