package bot_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bot Suite")
}

// ...
type TestCaseType string

const (
	HAPPY_PATH TestCaseType = "Happy"
	FAILURE    TestCaseType = "Failure"
)
