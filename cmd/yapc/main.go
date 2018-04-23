package main

import (
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/b-b3rn4rd/yapc/pkg/source"
	"github.com/b-b3rn4rd/yapc/pkg/storage"
	"github.com/sirupsen/logrus"
)

var (
	putJson     = kingpin.Command("put-json", "Puts the specified JSON file by creating Storage parameters.")
	getJson     = kingpin.Command("get-json", "Gets parameters from SSM in JSON format by specified path.")
	path        = kingpin.Flag("path", "SSM path").String()
	jsonFile    = putJson.Flag("json-file", "The path where your JSON file is located.").Required().ExistingFile()
	version     = "master"
	debug       = kingpin.Flag("debug", "Enable debug logging.").Short('d').Bool()
	stopOnError = kingpin.Flag("stop-on-error", "Stop export once error occurred.").Bool()
	logger      = logrus.New()
)

func main() {
	kingpin.Version(version)
	cmd := kingpin.Parse()
	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
		logger.SetLevel(logrus.DebugLevel)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := ssm.New(sess)

	strg := storage.New(svc, logger)

	switch cmd {
	case "get-json":
		_, err := strg.Export(*path)
		if err != nil {
			logrus.WithError(err).Fatal("big error")
		}

	case "put-json":
		j := source.SourceJSON{}
		r, err := os.Open(*jsonFile)
		if err != nil {
			logrus.Fatal("Big error")
		}
		defer r.Close()

		body, err := j.Flatten(r)
		if err != nil {
			logrus.Error(err)
		}

		err = strg.Import(body, *stopOnError)
		if err != nil {
			logger.WithError(err).Fatal("big error")
		}

	}
}
