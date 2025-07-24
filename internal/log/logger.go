package log

import (
	"github.com/sirupsen/logrus"
)

type LogModule string

const (
	DBModule     LogModule = "db"
	FacadeModule LogModule = "facade"
	TlgModule    LogModule = "tlg"
	BotModule    LogModule = "bot"
	StreamModule LogModule = "stream"
	WebModule    LogModule = "web"
)

func GetLogger(module LogModule) *logrus.Entry {
	return logrus.WithField("module", module)
}

func Setup(level string) {
	ll, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.WithError(err).Errorf("can not parse log level %s. using default ...", level)
		return
	}
	logrus.Infof("setting log level to %s", ll)
	logrus.SetLevel(ll)
}
