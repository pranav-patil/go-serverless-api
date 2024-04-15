package env

import "os"

func IsLocalOrTestEnv() bool {
	stage := os.Getenv("STAGE")
	return stage == "" || stage == "testing"
}
