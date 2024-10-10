package bot

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type botProfiler struct {
	pl *logrus.Entry
}

func (p *botProfiler) record(start time.Time, name string) {
	p.pl.WithField("call", name).Info(time.Since(start))
}
func NewBotProfiler(profileFile string, docID int64, worker string) *botProfiler {
	profileLogger := logrus.New()
	profileLogger.SetLevel(logrus.FatalLevel)
	if profileFile != "" {
		if f, err := os.OpenFile(profileFile, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
			logrus.WithError(err).Errorf("error creating profile file at %s", profileFile)
		} else {
			profileLogger.SetLevel(logrus.DebugLevel)
			profileLogger.SetOutput(f)
		}
	}
	pl := profileLogger.WithField("media", docID).WithField("worker", worker)
	return &botProfiler{pl: pl}
}
