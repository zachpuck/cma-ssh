package util

import "os"

// AmRunningInTest returns true if this binary is running under ginkgo
func AmRunningInTest() bool {
	_, runningUnderLocalTest := os.LookupEnv("CMA_SSH_DEV")
	return runningUnderLocalTest
}
