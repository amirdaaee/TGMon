package facade_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestFacade(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Facade Suite")
}

const (
	HAPPY_PATH   string = "Happy"
	FAILURE_PATH string = "Failure"
)
