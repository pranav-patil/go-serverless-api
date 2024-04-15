package launchdarkly

type LaunchdarklyAPI interface {
	IsEnabled(flag, accountId string) bool
	IsBookmarkFeatureEnabled(accountId string) bool
}

//go:generate mockgen -destination mocks/launchdarkly_mock.go -package mocks . LaunchdarklyAPI
