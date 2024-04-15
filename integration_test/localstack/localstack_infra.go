package main

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb/model"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/pranav-patil/go-serverless-api/pkg/s3"
)

func main() {
	argument := os.Args[1]

	os.Setenv("BOOKMARKS_BUCKET", "integration-test-bookmarks")
	os.Setenv("BOOKMARKS_SUMMARY_BUCKET", "integration-test-bookmarks-summary")

	if strings.EqualFold(argument, "SETUP") {
		Setup()
	} else if strings.EqualFold(argument, "DESTROY") {
		Destroy()
	} else {
		log.Printf("Invalid argrument %v passed. Valid arguments are 'setup' and 'destroy'", argument)
	}
}

func initialize() (bookmarksBucket string, bookmarksSummaryBucket string,
	s3Client s3.S3Client, dynamodbClient dynamodb.DynamoDBClient, err error) {
	bookmarksBucket = os.Getenv("BOOKMARKS_BUCKET")
	if bookmarksBucket == "" {
		log.Printf("Environment variable BOOKMARKS_BUCKET is empty.")
		return "", "", nil, nil, err
	}

	bookmarksSummaryBucket = os.Getenv("BOOKMARKS_SUMMARY_BUCKET")
	if bookmarksSummaryBucket == "" {
		log.Printf("Environment variable BOOKMARKS_SUMMARY_BUCKET is empty.")
		return "", "", nil, nil, err
	}

	s3Client, err = s3.NewS3Client()
	if err != nil {
		log.Printf("Unable to initialize S3: %v\n", err)
		return "", "", nil, nil, err
	}

	dynamodbClient, err = dynamodb.NewDynamoDBClient()
	if err != nil {
		log.Printf("Unable to initialize DynamoDB: %v\n", err)
		return "", "", nil, nil, err
	}

	return bookmarksBucket, bookmarksSummaryBucket, s3Client, dynamodbClient, err
}

func Setup() {
	bookmarksBucket, bookmarksSummaryBucket, s3Client, dynamodbClient, err := initialize()
	if err != nil {
		return
	}

	err = s3Client.CreateBucket(bookmarksBucket, mockutil.AWSRegion)
	if err != nil && checkError(err, bookmarksBucket) {
		return
	}

	err = s3Client.CreateBucket(bookmarksSummaryBucket, mockutil.AWSRegion)
	if err != nil && checkError(err, bookmarksSummaryBucket) {
		return
	}

	_, err = dynamodbClient.CreateTableIfNotExists(&model.UserBookmarks{})
	if err != nil {
		log.Printf("Unable to create DynamoDB table UserBookmarks: %v\n", err)
		return
	}

	_, err = dynamodbClient.CreateTableIfNotExists(&model.BookmarkDistribution{})
	if err != nil {
		log.Printf("Unable to create DynamoDB table BookmarkDistribution: %v\n", err)
		return
	}
}

func Destroy() {
	bookmarksBucket, bookmarksSummaryBucket, s3Client, dynamodbClient, err := initialize()
	if err != nil {
		return
	}

	err = s3Client.DeleteBucket(bookmarksBucket)
	if err != nil && checkError(err, bookmarksBucket) {
		return
	}

	err = s3Client.DeleteBucket(bookmarksSummaryBucket)
	if err != nil && checkError(err, bookmarksSummaryBucket) {
		return
	}

	err = dynamodbClient.DeleteTable((new(model.BookmarkDistribution)).GetTableName())
	if err != nil {
		log.Printf("Unable to delete DynamoDB table BookmarkDistribution: %v\n", err)
		return
	}

	err = dynamodbClient.DeleteTable((new(model.UserBookmarks)).GetTableName())
	if err != nil {
		log.Printf("Unable to delete DynamoDB table UserBookmarks: %v\n", err)
		return
	}
}

func checkError(err error, bucketName string) bool {
	var bae *types.BucketAlreadyExists
	var baoby *types.BucketAlreadyOwnedByYou

	if errors.As(err, &bae) || errors.As(err, &baoby) {
		log.Printf("S3 bucket %v already exists in Region %v.\n", bucketName, mockutil.AWSRegion)
	} else {
		log.Printf("Couldn't create bucket %v in Region %v. Here's why: %v\n",
			bucketName, mockutil.AWSRegion, err)
		return true
	}
	return false
}
