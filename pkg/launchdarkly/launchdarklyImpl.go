package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/rs/zerolog/log"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	ld "gopkg.in/launchdarkly/go-server-sdk.v5"
)

type Launchdarkly struct {
	ldClient *ld.LDClient
}

type launchdarklySMSDKKey struct {
	APIKey string `json:"api_key"`
}

const (
	LaunchDarklyKey       string = "EMPROVISE_LD_SDK_KEY"
	LaunchDarklyKeyID     string = "common/config/launchdarkly.json"
	ClientInitWaitSeconds int    = 10
	BookmarkFeatureFlag   string = "bookmark-feature-enabled"
)

func NewLaunchDarklyClient() (*Launchdarkly, error) {
	if env.IsLocalOrTestEnv() {
		ldClient, err := mockutil.GetTestLaunchDarklyClient()
		if err != nil {
			log.Error().Msg("error in setting up test launch darkly client")
			return nil, err
		}
		return &Launchdarkly{ldClient}, nil
	}

	ldSDKKey, err := getLDKeyFromEnv()

	if err != nil {
		log.Warn().Msg("can't get SDK key from environment variable, try AWS SM")

		var secretString string
		secretString, err = getSecretString(LaunchDarklyKeyID)
		if err != nil {
			return nil, err
		}

		var sdkKey launchdarklySMSDKKey
		err = json.Unmarshal([]byte(secretString), &sdkKey)
		if err != nil {
			log.Error().Msg("error in unmarshalling sdk key")
			return nil, err
		}

		if sdkKey.APIKey == "" {
			return nil, fmt.Errorf("LD SDK key is empty")
		}
	}

	ldClient, err := ld.MakeCustomClient(ldSDKKey,
		ld.Config{Offline: false},
		time.Duration(ClientInitWaitSeconds)*time.Second)

	if err == nil {
		return &Launchdarkly{ldClient}, nil
	} else {
		return nil, err
	}
}

func getLDKeyFromEnv() (string, error) {
	ldSDKKey, ok := os.LookupEnv(LaunchDarklyKey)
	if !ok {
		return "", fmt.Errorf("environment variable " + LaunchDarklyKey + " is not set")
	}

	if ldSDKKey == "" {
		return "", fmt.Errorf("LD SDK key is empty")
	}

	return ldSDKKey, nil
}

func getSecretString(secretID string) (secret string, err error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", err
	}

	secretClient := secretsmanager.NewFromConfig(cfg)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}
	result, err := secretClient.GetSecretValue(context.TODO(), input)
	return *result.SecretString, err
}

func (p *Launchdarkly) IsEnabled(flag, accountId string) bool {
	enabled, err := p.ldClient.BoolVariation(flag, lduser.NewUser(accountId), false)
	if err != nil {
		log.Warn().Msg("fail to invoke BoolVariation()")
	}
	return enabled
}

func (p *Launchdarkly) IsBookmarkFeatureEnabled(accountId string) bool {
	return p.IsEnabled(BookmarkFeatureFlag, accountId)
}
