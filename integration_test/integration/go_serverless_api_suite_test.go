package integration

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIntegrationGoServerlessAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Go Serverless API IntegrationTest Suite")
}
