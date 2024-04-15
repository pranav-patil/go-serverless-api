package model

import (
	"fmt"
	"os"
	"time"

	"github.com/pranav-patil/go-serverless-api/pkg/env"
)

type BookmarkDistributionEntity interface {
	Entity
}

var defaultAppDistTableName = "bookmark_distribution"

type BookmarkDistribution struct {
	PK             string    `dynamodbav:"PK"`
	SK             string    `dynamodbav:"SK"`
	UserId         string    `dynamodbav:"userId,omitempty" partitionKey:"UID"`
	DeviceId       string    `dynamodbav:"deviceId,omitempty" sortKey:"DID"`
	Status         string    `dynamodbav:"status,omitempty"`        // Pending, Failed, Success
	StatusMessage  string    `dynamodbav:"statusMessage,omitempty"` // Download bookmarks, enable policy, UDM load
	StartTimestamp time.Time `dynamodbav:"startTs,omitempty"`
	EndTimestamp   time.Time `dynamodbav:"endTs"`
}

func (distrib *BookmarkDistribution) GetTableName() string {
	tableName := os.Getenv("BOOKMARK_DISTRIBUTION_TABLE_NAME")

	if tableName == "" && env.IsLocalOrTestEnv() {
		tableName = defaultAppDistTableName
	}

	return tableName
}

func (distrib *BookmarkDistribution) String() string {
	return fmt.Sprintf("Status: %v\n\tStatusMessage: %v\n\tStartTs: %v\n\tEndTs: %v\n\tUserId: %v\n\tDeviceId: %v\n",
		distrib.Status, distrib.StatusMessage,
		distrib.StartTimestamp, distrib.EndTimestamp,
		distrib.UserId, distrib.DeviceId)
}
