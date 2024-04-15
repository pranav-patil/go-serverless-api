package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pranav-patil/go-serverless-api/func/api/helpers"
	"github.com/pranav-patil/go-serverless-api/func/api/middleware"
	"github.com/pranav-patil/go-serverless-api/func/api/models"
	"github.com/pranav-patil/go-serverless-api/pkg/constant"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb/model"
	"github.com/pranav-patil/go-serverless-api/pkg/s3"
	"github.com/pranav-patil/go-serverless-api/pkg/service"
	"github.com/pranav-patil/go-serverless-api/pkg/stepfunc"
	"github.com/pranav-patil/go-serverless-api/pkg/util"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

const (
	YYYYMMDDHHMMSS              = "20060102150405"
	YYYYMMDDHHMMSSSSS           = "20060102.150405.002"
	SignedURLExpirationSecs     = 300
	DistributionBookmarksLocked = "BookmarksLocked"
)

var (
	NewDynamoDBClient     = dynamodb.NewDynamoDBClient
	NewStepFunctionClient = stepfunc.NewStepFunctionClient

	NewExtServiceAPI = service.NewExternalService
	TimeNow          = time.Now
)

type DownloadBookmarksInput struct {
	UserId           string `json:"userId"`
	OperationId      string `json:"operationId"`
	DeviceId         string `json:"deviceId"`
	InstanceId       string `json:"instanceId"`
	Enabled          bool   `json:"enabled"`
	BookmarksVersion string `json:"bookmarksVersion"`
	Checksum         string `json:"fileChkSum"`
	S3PresignedURL   string `json:"presignedUrl"`
	RespToken        string `json:"respToken"`
}

func DistributeBookmarks(context *gin.Context) {
	request := models.DistributeBookmarksRequest{}

	if err := context.BindJSON(&request); err != nil {
		log.Debug().Msg("No device IDs passed for distribute request; defaults to all devices")
	}

	userId := context.GetString(middleware.UserIDCxt)
	dynamodbClient, err := NewDynamoDBClient()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	distribution := helpers.GetBookmarkByUser(dynamodbClient, userId)
	if errMsg := validateDistribution(distribution); errMsg != "" {
		context.JSON(http.StatusForbidden, gin.H{"error": errMsg})
		return
	}

	jwt := context.GetString(middleware.JWTToken)

	deviceMap, invalidDeviceIds, err := validateDevices(jwt, userId, request.DeviceIDs)
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}
	if isDevicesValidationError(context, userId, deviceMap, invalidDeviceIds) {
		return
	}

	s3Client, err := NewS3Client()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	distEntryPath := helpers.GetUserBookmarksS3Path(distribution)
	packageEntryPath := strings.ReplaceAll(distEntryPath, fmt.Sprintf("/%s/", userId), fmt.Sprintf("/%s/", util.MD5Hash(userId)))
	ipPackageBucketName := os.Getenv("BOOKMARKS_SUMMARY_BUCKET")

	err = updateDistributionStatus(dynamodbClient, distribution, DistributionBookmarksLocked)
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	var preSignedURL, checksum string

	if !strings.HasSuffix(distribution.LatestVersion, helpers.DeletedVersionSuffix) {
		checksum, err = getBookmarksAndCreatePackage(s3Client, distribution.LatestVersion, distEntryPath, packageEntryPath)
		if err != nil {
			helpers.SendInternalError(context, err)
			_ = updateDistributionStatus(dynamodbClient, distribution, constant.Failed)
			return
		}

		preSignedURL, err = s3Client.NewSignedGetURL(ipPackageBucketName, packageEntryPath, SignedURLExpirationSecs)
		if err != nil {
			helpers.SendCustomErrorMessage(context, http.StatusInternalServerError, "error in creating presigned url", err)
			_ = updateDistributionStatus(dynamodbClient, distribution, constant.Failed)
			return
		}
	}

	distributionJobList, err := addDistributionJobs(dynamodbClient, deviceMap, request.DeviceIDs,
		distribution, preSignedURL, checksum)

	if err != nil {
		helpers.SendInternalError(context, err)
		_ = updateDistributionStatus(dynamodbClient, distribution, constant.Failed)
		return
	}

	response := models.DistributedBookmarksResponse{DistributionJobList: distributionJobList}
	context.JSON(http.StatusOK, &response)
}

func GetDistributedBookmarks(context *gin.Context) {
	dynamodbClient, err := NewDynamoDBClient()
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}

	userId := context.GetString(middleware.UserIDCxt)
	distribution := helpers.GetBookmarkByUser(dynamodbClient, userId)
	if distribution == nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "no distributions found"})
		return
	}

	deviceDistribution := &model.BookmarkDistribution{UserId: userId}

	result, err := dynamodbClient.GetRecordsByKeyAndFields(deviceDistribution)
	if err != nil {
		helpers.SendInternalError(context, err)
		return
	}
	appDistributions := result.([]model.BookmarkDistribution)
	if len(appDistributions) == 0 {
		context.JSON(http.StatusNotFound, gin.H{"error": "no device distributions found"})
		return
	}

	var distributionJobList []models.WebCrawlerJob
	var shouldUpdateDistribution bool

	for index := range appDistributions {
		distrib := appDistributions[index]

		var deviceId int
		deviceId, err = strconv.Atoi(distrib.DeviceId)
		if err != nil {
			continue
		}

		var updated bool
		updated, err = updateDelayedApplainceDistributionToTimeout(dynamodbClient, &distrib)
		if err != nil {
			log.Error().Msgf("Dynamodb DeviceDistribution update failed for UserId %v, DeviceId %v: %v",
				userId, deviceId, err.Error())
			continue
		}
		if !shouldUpdateDistribution {
			shouldUpdateDistribution = updated
		}

		distributionJobList = append(distributionJobList, models.WebCrawlerJob{
			ID:             strconv.FormatInt(distribution.OperationId, 10),
			DeviceId:       deviceId,
			PackageVersion: distribution.LatestVersion,
			State:          distrib.Status,
			StatusMessage:  distrib.StatusMessage,
			StartTime:      distrib.StartTimestamp,
			EndTime:        distrib.EndTimestamp,
		})
	}

	if shouldUpdateDistribution && distribution.Status == constant.Timeout {
		err = dynamodbClient.UpdateRecordsByKey(distribution)
		if err != nil {
			log.Error().Msgf("Distribution update failed for UserId %s: %v", userId, err.Error())
		}
	}

	response := models.DistributedBookmarksResponse{
		DistributionJobList: distributionJobList,
		TotalCount:          len(distributionJobList),
	}
	context.JSON(http.StatusOK, &response)
}

func updateDelayedApplainceDistributionToTimeout(dynamodbClient dynamodb.DynamoDBClient,
	appDistribution *model.BookmarkDistribution) (bool, error) {
	if appDistribution != nil && appDistribution.Status == constant.Pending {
		currentTime := TimeNow()
		difference := (currentTime.Sub(appDistribution.StartTimestamp)).Minutes()

		if difference > helpers.DelayedStatusTimeInMinutes {
			appDistribution.Status = constant.Timeout
			appDistribution.StatusMessage = fmt.Sprintf("Setting to timeout after %v mins", difference)
			appDistribution.EndTimestamp = currentTime

			log.Debug().Msgf("Set DeviceDistribution status for userId %s to timeout after %v mins of pending status.",
				appDistribution.UserId, difference)

			return true, dynamodbClient.UpdateRecordsByKey(appDistribution)
		}
	}
	return false, nil
}

func addDistributionJobs(dynamodbClient dynamodb.DynamoDBClient, appMap map[int]string,
	requestDeviceIds []int, distribution *model.UserBookmarks, preSignedURL, checksum string) ([]models.WebCrawlerJob, error) {
	var distributionJobList []models.WebCrawlerJob

	currentTime := TimeNow()
	downloadBookmarksSfn := os.Getenv("DISTRIBUTION_STATE_MACHINE_ARN")

	if len(requestDeviceIds) == 0 {
		requestDeviceIds = make([]int, 0, len(appMap))
		for k := range appMap {
			requestDeviceIds = append(requestDeviceIds, k)
		}
	}

	sfnClient, err := NewStepFunctionClient()
	if err != nil {
		return nil, err
	}

	jobId, err := strconv.Atoi(currentTime.UTC().Format(YYYYMMDDHHMMSS))
	if err != nil {
		return nil, err
	}

	for _, device := range requestDeviceIds {
		deviceId := strconv.Itoa(device)

		appDistribution := &model.BookmarkDistribution{
			Status:         constant.Pending,
			StartTimestamp: currentTime,
			EndTimestamp:   time.Time{},
			UserId:         distribution.UserId,
			DeviceId:       deviceId,
		}

		err = dynamodbClient.AddRecord(appDistribution)
		if err != nil {
			return nil, err
		}

		downloadBookmarks := DownloadBookmarksInput{
			UserId:           distribution.UserId,
			OperationId:      strconv.FormatInt(int64(jobId), 10),
			DeviceId:         deviceId,
			InstanceId:       appMap[device],
			Enabled:          distribution.SyncEnabled,
			BookmarksVersion: distribution.LatestVersion,
			Checksum:         checksum,
			S3PresignedURL:   preSignedURL,
		}

		err = sfnClient.StartExecution(downloadBookmarksSfn, uuid.NewString(), downloadBookmarks)
		if err != nil {
			return nil, err
		}

		distributionJobList = append(distributionJobList, models.WebCrawlerJob{
			ID:             strconv.FormatInt(int64(jobId), 10),
			DeviceId:       device,
			PackageVersion: distribution.LatestVersion,
			State:          constant.Pending,
			StatusMessage:  "Distribution Process Triggered",
			StartTime:      TimeNow(),
			EndTime:        time.Time{},
		})
	}

	distribution.OperationId = int64(jobId)
	distribution.Status = constant.Pending
	distribution.StartTimestamp = currentTime
	distribution.EndTimestamp = time.Time{}
	distribution.ModifiedBookmarks = false

	err = dynamodbClient.UpdateRecordsByKey(distribution)
	if err != nil {
		return nil, err
	}

	return distributionJobList, nil
}

func validateDevices(jwt, userId string, requestDeviceIds []int) (deviceMap map[int]string,
	invalidDeviceIds []int, err error) {
	extServiceAPI, err := NewExtServiceAPI()
	if err != nil {
		return nil, nil, err
	}

	deviceList, err := extServiceAPI.GetUserDevices(jwt, userId)
	if err != nil {
		return nil, nil, err
	}

	var deviceIds []int
	deviceMap = make(map[int]string)

	for _, device := range deviceList.Devices {
		id := int(device.Id)
		deviceIds = append(deviceIds, id)
		deviceMap[id] = device.InstanceId
	}

	if len(requestDeviceIds) > 0 {
		for _, reqDeviceId := range requestDeviceIds {
			if !slices.Contains(deviceIds, reqDeviceId) {
				invalidDeviceIds = append(invalidDeviceIds, reqDeviceId)
			}
		}
	}

	return deviceMap, invalidDeviceIds, err
}

func getBookmarksAndCreatePackage(s3Client s3.S3Client, distVersion, distEntryPath, packageEntryPath string) (string, error) {
	var bookmarksBuilder strings.Builder

	bookmarksBucketName := os.Getenv("BOOKMARKS_BUCKET")

	data, err := s3Client.GetObject(bookmarksBucketName, distEntryPath)
	if err != nil {
		return "", err
	}

	bookmarks := models.BookmarkList{}
	err = json.Unmarshal(data, &bookmarks)
	if err != nil {
		return "", err
	}

	for i, p := range bookmarks.BookmarkEntry {
		if i > 0 {
			bookmarksBuilder.WriteString("\n")
		}

		err = util.ValidateURL(p.URL)
		if err != nil {
			log.Debug().Msgf("Invalid entry: %v", p.URL)
		} else {
			bookmarksBuilder.WriteString(p.URL)
		}
	}

	files := map[string]string{"version": distVersion, "ip-filtering.pkg": bookmarksBuilder.String()}
	content, err := util.CreateTarFile(files)
	if err != nil {
		return "", err
	}

	// Gzip the tar buffer
	compressedContent, err := util.ByteCompress(content)
	if err != nil {
		return "", err
	}

	// Write out the .tar.gz buffer to the Package S3 bucket
	ipPackageBucketName := os.Getenv("BOOKMARKS_SUMMARY_BUCKET")
	log.Debug().Msgf("Package bucket, key: %v, %v", ipPackageBucketName, packageEntryPath)

	err = s3Client.PutObject(ipPackageBucketName, packageEntryPath, JSON, "none", &compressedContent)
	if err != nil {
		return "", err
	}

	return util.SHA1Checksum(compressedContent), nil
}

func validateDistribution(distribution *model.UserBookmarks) string {
	if distribution == nil {
		return "no Bookmarks found to distribute"
	} else if helpers.IsDistributionPending(distribution) {
		return "distribution is in Progress"
	} else {
		return ""
	}
}

func updateDistributionStatus(dynamodbClient dynamodb.DynamoDBClient, distribution *model.UserBookmarks, status string) error {
	if status == DistributionBookmarksLocked || status == constant.Pending {
		distribution.StartTimestamp = TimeNow()
	} else {
		distribution.EndTimestamp = TimeNow()
	}
	err := dynamodbClient.UpdateRecordsByKey(distribution)
	if err != nil {
		log.Error().Msgf("failed to update distribution status %s: %v", status, err)
	}
	return err
}

func isDevicesValidationError(context *gin.Context, userId string, deviceMap map[int]string, invalidDeviceIds []int) bool {
	if len(deviceMap) == 0 {
		helpers.SendCustomErrorMessage(context, http.StatusBadRequest, "No devices found for the user",
			fmt.Errorf("external service get user devices returned empty list for user %s", userId))
		return true
	}

	if len(invalidDeviceIds) > 0 {
		context.JSON(http.StatusBadRequest, &models.InvalidDistributeBookmarksResponse{
			Error:     "Device Ids are not valid",
			DeviceIDs: invalidDeviceIds},
		)
		return true
	}
	return false
}
