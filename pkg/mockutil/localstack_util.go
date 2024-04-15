package mockutil

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/rs/zerolog/log"
	ld "gopkg.in/launchdarkly/go-server-sdk.v5"
	"gopkg.in/launchdarkly/go-server-sdk.v5/ldcomponents"
	"gopkg.in/launchdarkly/go-server-sdk.v5/ldfiledata"
	"gopkg.in/launchdarkly/go-server-sdk.v5/ldfilewatch"
)

const (
	AWSRegion   = "us-west-1"
	AWSEndPoint = "http://localhost:4566"
	LDFilename  = "./integration_test/localstack/ld-flags.json"
	WaitSeconds = 5
)

func GetLocalStackConfig() (aws.Config, error) {
	return config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(AWSRegion),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
							PartitionID:       "aws",
							URL:               AWSEndPoint,
							SigningRegion:     AWSRegion,
							HostnameImmutable: true,
						},
						nil
				})),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "dummy")),
	)
}

func GetTestLaunchDarklyClient() (*ld.LDClient, error) {
	log.Info().Msg("creating mock launch darkly client...")

	ldConfig := ld.Config{
		DataSource: ldfiledata.DataSource().
			FilePaths(LDFilename).Reloader(ldfilewatch.WatchFiles),
		Events: ldcomponents.NoEvents(),
	}
	return ld.MakeCustomClient("test-key", ldConfig, WaitSeconds*time.Second)
}
