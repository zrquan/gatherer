package output

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(os.Stdout)
}

func SetFormatter(json bool) {
	if json {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			TimestampFormat:           "2006-01-02 15:04:05",
			ForceColors:               true,
			EnvironmentOverrideColors: true,
		})
	}
}
