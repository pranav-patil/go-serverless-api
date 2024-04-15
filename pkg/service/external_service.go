package service

type ExternalServiceAPI interface {
	GetUserDevices(string, string) (*DeviceListResponse, error)
}

//go:generate mockgen -destination mocks/external_service_mock.go -package mocks . ExternalServiceAPI
