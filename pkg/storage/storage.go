package storage

import (
	"sync"

	"fmt"

	"strings"

	"reflect"

	"strconv"

	"time"

	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
)

type Storage interface {
	Import(map[string]interface{}) (int16, error)
	Export(string) (interface{}, error)
	Delete(map[string]interface{}) (int16, error)
}

type SSMStorage struct {
	svc    ssmiface.SSMAPI
	logger *logrus.Logger
	sleep  int
}

func New(svc ssmiface.SSMAPI, logger *logrus.Logger) *SSMStorage {
	return &SSMStorage{
		svc:    svc,
		logger: logger,
		sleep:  10,
	}
}

func (s *SSMStorage) Export(path string, decrypt bool) (interface{}, error) {
	values := map[string]interface{}{}
	mx := sync.Mutex{}
	s.logger.WithField("path", path).Debug("get parameters by path")

	var wg sync.WaitGroup
	var i uint32

	bar := pb.New(0)
	bar.Output = os.Stderr
	bar.Start()

	err := s.svc.GetParametersByPathPages(&ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(decrypt),
	}, func(page *ssm.GetParametersByPathOutput, lastPage bool) bool {
		bar.SetTotal(int(bar.Total) + len(page.Parameters))

		for _, p := range page.Parameters {
			wg.Add(1)

			if i%20 == 0 && i > 0 {
				s.logger.Debugf("sleep for a %d seconds", s.sleep)
				time.Sleep(time.Duration(s.sleep) * time.Second)
			}

			i++
			go func(name string, value string) {
				defer func() {
					bar.Increment()
					wg.Done()
				}()

				s.logger.WithField("name", name).Debug("getting parameter type")
				resp, err := s.svc.ListTagsForResource(&ssm.ListTagsForResourceInput{
					ResourceType: aws.String(ssm.ResourceTypeForTaggingParameter),
					ResourceId:   aws.String(name),
				})

				if err != nil {
					s.logger.WithField("name", name).WithError(err).Info("can't get parameter type use string")
					mx.Lock()
					values[name] = value
					mx.Unlock()
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

				mx.Lock()
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
				mx.Unlock()

			}(aws.StringValue(p.Name), aws.StringValue(p.Value))
		}

		return !lastPage
	})
	if err != nil {
		return nil, err
	}

	wg.Wait()
	bar.Finish()

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
		ksr := make([]string, len(ks))
		for ksi, ksv := range ks {
			ksr[len(ks)-(ksi+1)] = ksv
		}

		for _, kv := range ksr {
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

func (s *SSMStorage) Delete(values map[string]interface{}) (int, error) {
	var wg sync.WaitGroup
	var delParamError error
	var i uint32

	total := len(values)
	bar := pb.New(total)
	bar.Output = os.Stderr
	bar.Start()

	for k := range values {
		wg.Add(1)

		if i%20 == 0 && i > 0 {
			s.logger.Debugf("sleep for a %d seconds", s.sleep)
			time.Sleep(time.Duration(s.sleep) * time.Second)
		}

		i++

		go func(k string) {
			defer func() {
				bar.Increment()
				wg.Done()
			}()

			k = fmt.Sprintf("/%s", k)
			s.logger.WithField("name", k).Debug("deleting ssm parameter")

			_, err := s.svc.DeleteParameter(&ssm.DeleteParameterInput{
				Name: aws.String(k),
			})
			if err != nil {
				delParamError = err
			}

			s.logger.WithField("name", k).Debug("deleting metadata for ssm parameter")

			s.svc.RemoveTagsFromResource(&ssm.RemoveTagsFromResourceInput{
				ResourceId: aws.String(k),
			})

		}(k)
	}

	wg.Wait()
	bar.Finish()

	return total, delParamError
}

func (s *SSMStorage) Import(values map[string]interface{}, msg string, encrypt bool) (int, error) {
	var wg sync.WaitGroup
	var putParamError error
	var i uint32
	var paramType string

	if encrypt {
		paramType = ssm.ParameterTypeSecureString
	} else {
		paramType = ssm.ParameterTypeString
	}

	total := len(values)

	bar := pb.StartNew(total)
	bar.Output = os.Stderr

	for k, v := range values {
		wg.Add(1)

		if i%10 == 0 && i > 0 {
			s.logger.Debugf("sleep for a %d seconds", s.sleep)
			time.Sleep(time.Duration(s.sleep) * time.Second)
		}

		i++

		go func(k string, v interface{}) {
			defer func() {
				bar.Increment()
				wg.Done()
			}()
			k = fmt.Sprintf("/%s", k)
			s.logger.WithField("name", k).Debug("putting ssm parameter")

			_, err := s.svc.PutParameter(&ssm.PutParameterInput{
				Name:        aws.String(k),
				Value:       aws.String(fmt.Sprint(v)),
				Type:        aws.String(paramType),
				Overwrite:   aws.Bool(true),
				Description: aws.String(msg),
			})
			if err != nil {
				putParamError = err
				return
			}

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
	bar.Finish()

	return total, putParamError
}
