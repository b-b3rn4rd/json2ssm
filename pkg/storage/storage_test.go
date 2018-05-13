package storage_test

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/b-b3rn4rd/json2ssm/mocks"
	"github.com/b-b3rn4rd/json2ssm/pkg/storage"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type SSMMock struct {
	ssmiface.SSMAPI
	output                    *ssm.GetParametersByPathOutput
	listTagsForResourceOutput *ssm.ListTagsForResourceOutput
}

func (s *SSMMock) GetParametersByPathPages(input *ssm.GetParametersByPathInput, cb func(*ssm.GetParametersByPathOutput, bool) bool) error {
	cb(s.output, true)
	return nil
}
func (s *SSMMock) ListTagsForResource(input *ssm.ListTagsForResourceInput) (*ssm.ListTagsForResourceOutput, error) {
	return s.listTagsForResourceOutput, nil
}

func TestDelete(t *testing.T) {
	values := map[string]interface{}{
		"0/name":         "bernard",
		"0/address/work": "1 flinders",
		"0/address/home": "1 st kilda rd",
		"1/name":         "keith",
		"1/address/work": "2 flinders",
		"1/address/home": "2 st kilda rd",
	}
	s := &mocks.SSMAPI{}

	deleteParameterExpectedInput := map[string]*ssm.DeleteParameterInput{
		"/0/name": {
			Name: aws.String("/0/name"),
		},
		"/0/address/work": {
			Name: aws.String("/0/address/work"),
		},
		"/0/address/home": {
			Name: aws.String("/0/address/home"),
		},
		"/1/name": {
			Name: aws.String("/1/name"),
		},
		"/1/address/work": {
			Name: aws.String("/1/address/work"),
		},
		"/1/address/home": {
			Name: aws.String("/1/address/home"),
		},
	}

	removeTagsToResourceExpectedInput := map[string]*ssm.RemoveTagsFromResourceInput{
		"/0/name": {
			ResourceId: aws.String("/0/name"),
		},
		"/0/address/work": {
			ResourceId: aws.String("/0/address/work"),
		},
		"/0/address/home": {
			ResourceId: aws.String("/0/address/home"),
		},
		"/1/name": {
			ResourceId: aws.String("/1/name"),
		},
		"/1/address/work": {
			ResourceId: aws.String("/1/address/work"),
		},
		"/1/address/home": {
			ResourceId: aws.String("/1/address/home"),
		},
	}
	deleteParameterExpectedOutput := &ssm.DeleteParameterOutput{}
	removeTagsToResourceExpectedOutput := &ssm.RemoveTagsFromResourceOutput{}

	s.On("DeleteParameter", mock.MatchedBy(func(input *ssm.DeleteParameterInput) bool {
		v := deleteParameterExpectedInput[aws.StringValue(input.Name)]
		return assert.Equal(t, v, input)
	})).Return(deleteParameterExpectedOutput, nil)

	s.On("RemoveTagsFromResource", mock.MatchedBy(func(input *ssm.RemoveTagsFromResourceInput) bool {
		v := removeTagsToResourceExpectedInput[aws.StringValue(input.ResourceId)]
		return assert.Equal(t, v, input)
	})).Return(removeTagsToResourceExpectedOutput, nil)

	logger, _ := test.NewNullLogger()
	str := storage.New(s, logger)
	str.Delete(values)

	s.AssertNumberOfCalls(t, "DeleteParameter", 6)
	s.AssertNumberOfCalls(t, "RemoveTagsFromResource", 6)
}

func TestExport(t *testing.T) {
	s := &SSMMock{}
	s.output = &ssm.GetParametersByPathOutput{
		Parameters: []*ssm.Parameter{
			{
				Name:  aws.String("/0/name"),
				Type:  aws.String(ssm.ParameterTypeString),
				Value: aws.String("bernard"),
			},
			{
				Name:  aws.String("/0/address/work"),
				Type:  aws.String(ssm.ParameterTypeString),
				Value: aws.String("1 flinders"),
			},
			{
				Name:  aws.String("/0/address/home"),
				Type:  aws.String(ssm.ParameterTypeString),
				Value: aws.String("1 st kilda rd"),
			},
		},
	}
	s.listTagsForResourceOutput = &ssm.ListTagsForResourceOutput{TagList: []*ssm.Tag{
		{
			Key:   aws.String("type"),
			Value: aws.String("string"),
		},
	}}

	logger, _ := test.NewNullLogger()
	str := storage.New(s, logger)
	r, _ := str.Export("/0")

	expected := map[string]interface{}{
		"name": "bernard",
		"address": map[string]interface{}{
			"home": "1 st kilda rd",
			"work": "1 flinders",
		},
	}

	assert.Equal(t, expected, r)
}

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

	addTagsToResourceExpectedInput := map[string]*ssm.AddTagsToResourceInput{
		"/0/name": {
			ResourceId:   aws.String("/0/name"),
			ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
			Tags: []*ssm.Tag{&ssm.Tag{
				Key:   aws.String("type"),
				Value: aws.String("string"),
			}},
		},
		"/0/address/work": {
			ResourceId:   aws.String("/0/address/work"),
			ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
			Tags: []*ssm.Tag{&ssm.Tag{
				Key:   aws.String("type"),
				Value: aws.String("string"),
			}},
		},
		"/0/address/home": {
			ResourceId:   aws.String("/0/address/home"),
			ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
			Tags: []*ssm.Tag{&ssm.Tag{
				Key:   aws.String("type"),
				Value: aws.String("string"),
			}},
		},
		"/1/name": {
			ResourceId:   aws.String("/1/name"),
			ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
			Tags: []*ssm.Tag{&ssm.Tag{
				Key:   aws.String("type"),
				Value: aws.String("string"),
			}},
		},
		"/1/address/work": {
			ResourceId:   aws.String("/1/address/work"),
			ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
			Tags: []*ssm.Tag{&ssm.Tag{
				Key:   aws.String("type"),
				Value: aws.String("string"),
			}},
		},
		"/1/address/home": {
			ResourceId:   aws.String("/1/address/home"),
			ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
			Tags: []*ssm.Tag{&ssm.Tag{
				Key:   aws.String("type"),
				Value: aws.String("string"),
			}},
		},
	}
	putParameterExpectedOutput := &ssm.PutParameterOutput{}
	addTagsToResourceExpectedOutput := &ssm.AddTagsToResourceOutput{}

	s.On("PutParameter", mock.MatchedBy(func(input *ssm.PutParameterInput) bool {
		v := putParameterExpectedInput[aws.StringValue(input.Name)]
		return assert.Equal(t, v, input)
	})).Return(putParameterExpectedOutput, nil)

	s.On("AddTagsToResource", mock.MatchedBy(func(input *ssm.AddTagsToResourceInput) bool {
		v := addTagsToResourceExpectedInput[aws.StringValue(input.ResourceId)]
		return assert.Equal(t, v, input)
	})).Return(addTagsToResourceExpectedOutput, nil)

	logger, _ := test.NewNullLogger()
	str := storage.New(s, logger)
	str.Import(values)

	s.AssertNumberOfCalls(t, "PutParameter", 6)
	s.AssertNumberOfCalls(t, "AddTagsToResource", 6)
}
