package storage_test

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/b-b3rn4rd/yapc/mocks"
	"github.com/b-b3rn4rd/yapc/pkg/storage"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestImport(t *testing.T) {
	values := map[string]interface{}{
		"0/name":         "bernard",
		"0/address/work": "1 flinders",
		"0/address/home": "1 st kilda rd",
		"1/name":         "keith",
		"1/address/work": "2 flinders",
		"1/address/home": "2 st kilda rd",
	}
	s := &mocks.SSMAPI{}

	putParameterExpectedInput := map[string]*ssm.PutParameterInput{
		"/0/name": {
			Name:      aws.String("/0/name"),
			Value:     aws.String("bernard"),
			Type:      aws.String(ssm.ParameterTypeString),
			Overwrite: aws.Bool(true),
		},
		"/0/address/work": {
			Name:      aws.String("/0/address/work"),
			Value:     aws.String("1 flinders"),
			Type:      aws.String(ssm.ParameterTypeString),
			Overwrite: aws.Bool(true),
		},
		"/0/address/home": {
			Name:      aws.String("/0/address/home"),
			Value:     aws.String("1 st kilda rd"),
			Type:      aws.String(ssm.ParameterTypeString),
			Overwrite: aws.Bool(true),
		},
		"/1/name": {
			Name:      aws.String("/1/name"),
			Value:     aws.String("keith"),
			Type:      aws.String(ssm.ParameterTypeString),
			Overwrite: aws.Bool(true),
		},
		"/1/address/work": {
			Name:      aws.String("/1/address/work"),
			Value:     aws.String("2 flinders"),
			Type:      aws.String(ssm.ParameterTypeString),
			Overwrite: aws.Bool(true),
		},
		"/1/address/home": {
			Name:      aws.String("/1/address/home"),
			Value:     aws.String("2 st kilda rd"),
			Type:      aws.String(ssm.ParameterTypeString),
			Overwrite: aws.Bool(true),
		},
	}
	putParameterExpectedOutput := &ssm.PutParameterOutput{}

	s.On("PutParameter", mock.MatchedBy(func(input *ssm.PutParameterInput) bool {
		v := putParameterExpectedInput[aws.StringValue(input.Name)]
		return assert.Equal(t, v, input)
	})).Times(6).Return(putParameterExpectedOutput, nil)

	logger, _ := test.NewNullLogger()
	str := storage.New(s, logger)
	str.Import(values)
}
