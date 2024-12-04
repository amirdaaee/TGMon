package db_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Db Suite")
}

// ...
type TestCaseType string

const (
	HAPPY_PATH TestCaseType = "Happy"
	FAILURE    TestCaseType = "Failure"
)
