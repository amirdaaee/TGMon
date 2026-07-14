package facade_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

func TestFacade(t *testing.T) {
	zap.ReplaceGlobals(zap.NewNop())
	RegisterFailHandler(Fail)
	RunSpecs(t, "Facade Suite")
}

const (
	HAPPY_PATH   string = "Happy"
	FAILURE_PATH string = "Failure"
)
