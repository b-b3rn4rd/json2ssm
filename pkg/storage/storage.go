package storage

import (
	"sync"

	"context"

	"fmt"

	"strings"

	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/sirupsen/logrus"
)

type Importer interface {
	Import(map[string]string, bool) error
}

type Storage struct {
	svc    ssmiface.SSMAPI
	logger *logrus.Logger
}

func New(svc ssmiface.SSMAPI, logger *logrus.Logger) *Storage {
	return &Storage{
		svc:    svc,
		logger: logger,
	}
}

func (s *Storage) Export(path string) (map[string]interface{}, error) {
	values := map[string]string{}

	s.logger.WithField("path", path).Debug("get parameters by path")

	err := s.svc.GetParametersByPathPages(&ssm.GetParametersByPathInput{
		Path:      aws.String(path),
		Recursive: aws.Bool(true),
	}, func(page *ssm.GetParametersByPathOutput, lastPage bool) bool {
		for _, p := range page.Parameters {
			values[aws.StringValue(p.Name)] = aws.StringValue(p.Value)
		}

		return !lastPage
	})
	if err != nil {
		return nil, err
	}

	s.unflattern(values)
	return nil, nil
}

func (s *Storage) findParameterType(parameterName string) {
	resp, err := s.svc.ListTagsForResource(&ssm.ListTagsForResourceInput{
		ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
		ResourceId:   aws.String(parameterName),
	})
	if err != nil {

	}

	paramType := func() string {
		for _, tag := range resp.TagList {
			if *tag.Key == "type" {
				return aws.StringValue(tag.Value)
			}
		}

		return "string"
	}()
}

func (s *Storage) unflattern(v map[string]string) (map[string]interface{}, error) {
	var tree = make(map[string]interface{})
	for k, v := range v {
		ks := strings.Split(strings.TrimLeft(k, "/"), "/")
		tr := tree
		for _, tk := range ks[:len(ks)-1] {
			trnew, ok := tr[tk]
			if !ok {
				trnew = make(map[string]interface{})
				tr[tk] = trnew
			}
			tr = trnew.(map[string]interface{})
		}
		tr[ks[len(ks)-1]] = v
	}
	fmt.Println(tree)
	return tree, nil
}

func (s *Storage) Import(values map[string]interface{}, stopOnError bool) error {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var putParamError error
	for k, v := range values {
		wg.Add(1)
		go func(k string, v interface{}) {
			defer wg.Done()
			k = fmt.Sprintf("/%s", k)
			s.logger.WithField("name", k).Debug("putting ssm parameter")

			if putParamError != nil && stopOnError {
				s.logger.WithField("name", k).Debug("skipping set parameter, error occurred before")
				return
			}
			_, err := s.svc.PutParameterWithContext(ctx, &ssm.PutParameterInput{
				Name:      aws.String(k),
				Value:     aws.String(fmt.Sprint(v)),
				Type:      aws.String(ssm.ParameterTypeString),
				Overwrite: aws.Bool(true),
			})
			if err != nil {
				putParamError = err
			}
			_, err = s.svc.AddTagsToResourceWithContext(ctx, &ssm.AddTagsToResourceInput{
				ResourceId:   aws.String(k),
				ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
				Tags: []*ssm.Tag{&ssm.Tag{
					Key:   aws.String("type"),
					Value: aws.String(reflect.TypeOf(v).Kind().String()),
				}},
			})
			if err != nil {
				putParamError = err
			}

		}(k, v)
	}

	wg.Wait()

	return putParamError
}
