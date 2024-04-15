package model

import (
	"fmt"
	"os"
	"time"

	"github.com/pranav-patil/go-serverless-api/pkg/env"
)

type UserBookmarksEntity interface {
	Entity
}

const defaultDistTableName = "user_bookmarks"

type UserBookmarks struct {
	PK                string    `dynamodbav:"PK"`
	UserId            string    `dynamodbav:"userId,omitempty" partitionKey:"UID"`
	OperationId       int64     `dynamodbav:"operationId,omitempty"`
	Status            string    `dynamodbav:"status,omitempty"`
	StartTimestamp    time.Time `dynamodbav:"startTs,omitempty"`
	EndTimestamp      time.Time `dynamodbav:"endTs,omitempty"`
	SyncEnabled       bool      `dynamodbav:"syncEnabled"`
	LatestVersion     string    `dynamodbav:"latestVersion,omitempty"`
	ModifiedBookmarks bool      `dynamodbav:"modifiedBookmarks"`
}

func (userBookmarks *UserBookmarks) GetTableName() string {
	tableName := os.Getenv("USER_BOOKMARK_TABLE_NAME")

	if tableName == "" && env.IsLocalOrTestEnv() {
		tableName = defaultDistTableName
	}

	return tableName
}

func (userBookmarks *UserBookmarks) String() string {
	return fmt.Sprintf(
		"UserId: %v\n\tEnabled: %v\n\tLatestVersion: %v\n\tModifiedBookmarks: %v\n",
		userBookmarks.UserId, userBookmarks.SyncEnabled,
		userBookmarks.LatestVersion, userBookmarks.ModifiedBookmarks)
}
