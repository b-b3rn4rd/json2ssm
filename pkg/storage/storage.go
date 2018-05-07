package storage

import (
	"sync"

	"fmt"

	"strings"

	"reflect"

	"strconv"

	"sync/atomic"

	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/sirupsen/logrus"
)

type Storage interface {
	Import(map[string]interface{}) (int16, error)
	Export(string) (interface{}, error)
	Delete(map[string]interface{}) (int16, error)
}

type SSMStorage struct {
	svc    ssmiface.SSMAPI
	logger *logrus.Logger
}

func New(svc ssmiface.SSMAPI, logger *logrus.Logger) *SSMStorage {
	return &SSMStorage{
		svc:    svc,
		logger: logger,
	}
}

func (s *SSMStorage) Export(path string) (interface{}, error) {
	values := map[string]interface{}{}
	s.logger.WithField("path", path).Debug("get parameters by path")

	var wg sync.WaitGroup
	var i uint32

	err := s.svc.GetParametersByPathPages(&ssm.GetParametersByPathInput{
		Path:      aws.String(path),
		Recursive: aws.Bool(true),
	}, func(page *ssm.GetParametersByPathOutput, lastPage bool) bool {
		for _, p := range page.Parameters {
			wg.Add(1)

			if i%10 == 0 && i > 0 {
				s.logger.Debug("sleep for a 15 seconds")
				time.Sleep(15 * time.Second)
			}

			i++
			go func(name string, value string) {
				defer wg.Done()

				s.logger.WithField("name", name).Debug("getting parameter type")
				resp, err := s.svc.ListTagsForResource(&ssm.ListTagsForResourceInput{
					ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
					ResourceId:   aws.String(name),
				})
				if err != nil {
					s.logger.WithField("name", name).WithError(err).Debug("can't get parameter type use string")
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

	tree := make(map[string]interface{})

	for k, v := range values {
		tree[strings.TrimPrefix(k, path)] = v
	}

	return s.unflattern(tree)
}

func (s *SSMStorage) unflattern(params map[string]interface{}) (interface{}, error) {
	var mergeMaps func(m1 interface{}, m2 interface{}) interface{}
	mergeMaps = func(m1 interface{}, m2 interface{}) interface{} {

		switch m2 := m2.(type) {
		case string:
			return m2
		case float64:
			return m2
		case nil:
			return m2
		case []interface{}:
			m1, _ := m1.([]interface{})
			for i2, v2 := range m2 {
				if v2 == nil {
					continue
				}
				for {
					if len(m1) >= (i2 + 1) {
						break
					}
					m1 = append(m1, "")
				}
				m1[i2] = mergeMaps(m1[i2], v2)

			}
			return m1
		case map[string]interface{}:

			m1, ok := m1.(map[string]interface{})
			if !ok {

				return m2
			}
			for k2, v2 := range m2 {
				if v1, ok := m1[k2]; ok {

					m1[k2] = mergeMaps(v1, v2)

				} else {
					m1[k2] = v2
				}
			}
		}

		return m1
	}

	var tree interface{}

	for k, v := range params {
		ks := strings.Split(strings.TrimPrefix(k, "/"), "/")
		ks_r := make([]string, len(ks))
		for ksi, ksv := range ks {
			ks_r[len(ks)-(ksi+1)] = ksv
		}

		for _, kv := range ks_r {
			if ik, err := strconv.Atoi(kv); err == nil {

				tmp := make([]interface{}, (ik + 1))
				tmp[ik] = v
				v = tmp
				continue
			}
			v = map[string]interface{}{kv: v}
		}
		tree = mergeMaps(tree, v)
	}

	return tree, nil
}

func (s *SSMStorage) Delete(values map[string]interface{}) (int16, error) {
	var wg sync.WaitGroup
	var delParamError error
	var total int16
	for k, _ := range values {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()

			k = fmt.Sprintf("/%s", k)
			s.logger.WithField("name", k).Debug("deleting ssm parameter")

			_, err := s.svc.DeleteParameter(&ssm.DeleteParameterInput{
				Name: aws.String(k),
			})
			if err != nil {
				delParamError = err
			}

			total++

			s.logger.WithField("name", k).Debug("deleting metadata for ssm parameter")

			s.svc.RemoveTagsFromResource(&ssm.RemoveTagsFromResourceInput{
				ResourceId: aws.String(k),
			})

		}(k)
	}
	wg.Wait()

	return total, delParamError
}

func (s *SSMStorage) Import(values map[string]interface{}) (uint32, error) {
	var wg sync.WaitGroup
	var total uint32
	var putParamError error

	var i uint32

	for k, v := range values {
		wg.Add(1)

		if i%10 == 0 && i > 0 {
			s.logger.Debug("sleep for a minute")
			time.Sleep(15 * time.Second)
		}

		i++

		go func(k string, v interface{}) {
			defer wg.Done()
			k = fmt.Sprintf("/%s", k)
			s.logger.WithField("name", k).Debug("putting ssm parameter")

			_, err := s.svc.PutParameter(&ssm.PutParameterInput{
				Name:      aws.String(k),
				Value:     aws.String(fmt.Sprint(v)),
				Type:      aws.String(ssm.ParameterTypeString),
				Overwrite: aws.Bool(true),
			})
			if err != nil {
				putParamError = err
				return
			}

			atomic.AddUint32(&total, 1)

			_, err = s.svc.AddTagsToResource(&ssm.AddTagsToResourceInput{
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

	return atomic.LoadUint32(&total), putParamError
}
