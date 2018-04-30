package storage

import (
	"sync"

	"context"

	"fmt"

	"strings"

	"reflect"

	"strconv"

	"encoding/json"

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
	values := map[string]interface{}{}
	s.logger.WithField("path", path).Debug("get parameters by path")

	var wg sync.WaitGroup

	err := s.svc.GetParametersByPathPages(&ssm.GetParametersByPathInput{
		Path:      aws.String(path),
		Recursive: aws.Bool(true),
	}, func(page *ssm.GetParametersByPathOutput, lastPage bool) bool {
		for _, p := range page.Parameters {
			wg.Add(1)
			go func(name string, value string) {
				defer wg.Done()

				s.logger.WithField("name", name).Debug("getting parameter type")
				resp, err := s.svc.ListTagsForResource(&ssm.ListTagsForResourceInput{
					ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
					ResourceId:   aws.String(name),
				})
				if err != nil {
					s.logger.WithField("name", name).Debug("can't get parameter type use string")
					values[name] = value
				}

				vType := func() string {
					for _, tag := range resp.TagList {
						if *tag.Key == "type" {
							return aws.StringValue(tag.Value)
						}
					}

					return "string"
				}()

				s.logger.WithField("name", name).Debugf("converting to %s", vType)

				switch vType {
				case "bool":
					values[name], _ = strconv.ParseBool(value)
				case "float64":
					values[name], _ = strconv.ParseFloat(value, 64)
				case "nil":
					values[name] = nil
				default:
					values[name] = value
				}

			}(aws.StringValue(p.Name), aws.StringValue(p.Value))
			values[aws.StringValue(p.Name)] = values
		}

		return !lastPage
	})
	if err != nil {
		return nil, err
	}

	wg.Wait()
	s.unflattern(values)
	return nil, nil
}

//func (s *Storage) unflatternSlice(tr interface{}, tk string) []interface{} {
//	trnew, ok := tr[tk]
//	if !ok {
//		trnew = make([]interface{}, 0)
//
//		tr[tk] = trnew
//	}
//
//	return trnew.([]interface{})
//}
//
//func (s *Storage) unflatternMap(tr interface{}, tk string) map[string]interface{} {
//	trnew, ok := tr[tk]
//	if !ok {
//		trnew = make(map[string]interface{})
//		tr[tk] = trnew
//	}
//
//	return trnew.(map[string]interface{})
//}

func (s *Storage) unflattern(v map[string]interface{}) (map[string]interface{}, error) {
	var tree interface{}
	for k, v := range v {
		ks := strings.Split(strings.TrimLeft(k, "/"), "/")
		tr := tree
		for i, tk := range ks[:len(ks)-1] {
			var trnew interface{}
			ok := true
			// check if tk is numeric
			if _, err := strconv.Atoi(tk); err != nil {
				s.logger.WithField("key", tk).Debug("is not interger")
				if _, ok = tr.(map[string]interface{}); !ok {
					s.logger.WithField("key", tk).Debug("tr is empty")
					tr = map[string]interface{}{}
				}
				trnew, ok = tr.(map[string]interface{})[tk]
			}

			if !ok {
				trnew = make(map[string]interface{})

				if len(ks) > i+1 {
					if _, err := strconv.Atoi(ks[i+1]); err == nil {
						trnew = make([]interface{}, 0)
					}
				}

				tr.(map[string]interface{})[tk] = trnew
			}

			tr = trnew
		}
		switch tr.(type) {
		case map[string]interface{}:
			tr.(map[string]interface{})[ks[len(ks)-1]] = v
		case []interface{}:
			tr = append(tr.([]interface{}), v)
		}

	}
	fmt.Println(tree)
	raw, _ := json.MarshalIndent(tree, "", " ")
	fmt.Print(string(raw))
	return nil, nil
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
