package main

import (
	"os"

	"encoding/json"
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/b-b3rn4rd/json2ssm/pkg/source"
	"github.com/b-b3rn4rd/json2ssm/pkg/storage"
	"github.com/sirupsen/logrus"
)

var (
	putJson     = kingpin.Command("put-json", "Creates SSM parameters from the specified JSON file.")
	getJson     = kingpin.Command("get-json", "Retrieves JSON document from SSM parameter store using given path (prefix).")
	delJson     = kingpin.Command("del-json", "Deletes parameters from SSM parameter store based on the specified JSON file.")
	getPath     = getJson.Flag("path", "SSM parameter store path (prefix)").Required().String()
	putJsonFile = putJson.Flag("json-file", "The path where your JSON file is located.").Required().ExistingFile()
	delJsonFile = delJson.Flag("json-file", "The path where your JSON file is located.").Required().ExistingFile()
	version     = "master"
	debug       = kingpin.Flag("debug", "Enable debug logging.").Short('d').Bool()
	logger      = logrus.New()
	writer      = os.Stdout
)

func main() {
	kingpin.Version(version)
	cmd := kingpin.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
		logger.SetLevel(logrus.DebugLevel)
	}

	logger.Formatter = &logrus.JSONFormatter{}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	strg := storage.New(ssm.New(sess), logger)

	switch cmd {

	case "del-json":
		j := source.SourceJSON{}
		r, err := os.Open(*delJsonFile)
		if err != nil {
			logrus.WithError(err).Fatal("error while opening file")
		}
		defer r.Close()

		body, err := j.Flatten(r)
		if err != nil {
			logrus.WithError(err).Fatal("error while flattering")
		}

		total, err := strg.Delete(body)
		if err != nil {
			logger.WithError(err).Fatal("error while deleting")
		}

		fmt.Fprintf(writer, "\nDeletion has successfully finished, %d parameters have been removed from SSM parameter store. \n", total)

	case "get-json":
		values, err := strg.Export(*getPath)
		if err != nil {
			logrus.WithError(err).Fatal("error while exporting")
		}
		raw, _ := json.MarshalIndent(values, "", " ")
		fmt.Fprint(writer, string(raw))

	case "put-json":
		j := source.SourceJSON{}
		r, err := os.Open(*putJsonFile)
		if err != nil {
			logrus.WithError(err).Fatal("error while opening file")
		}
		defer r.Close()

		body, err := j.Flatten(r)
		if err != nil {
			logrus.WithError(err).Fatal("error while flattering")
		}

		total, err := strg.Import(body)
		if err != nil {
			logrus.WithError(err).Fatal("error while importing")
		}

		fmt.Fprintf(writer, "\nImport has successfully finished, %d parameters have been (over)written to SSM parameter store. \n", total)
	}
}
