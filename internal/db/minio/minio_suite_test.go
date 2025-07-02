package minio_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMinio(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Minio Suite")
}

type TestCaseType string

const (
	HAPPY_PATH TestCaseType = "Happy"
	FAILURE    TestCaseType = "Failure"
)
