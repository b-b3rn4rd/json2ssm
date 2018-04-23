package source

import (
	"encoding/json"
	"io"

	"io/ioutil"

	"github.com/nytlabs/gojsonexplode"
)

type SourceJSON struct{}

func (j *SourceJSON) Flatten(r io.Reader) (map[string]interface{}, error) {
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	json_raw, err := gojsonexplode.Explodejson(raw, "/")
	var v map[string]interface{}

	err = json.Unmarshal(json_raw, &v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (j *SourceJSON) unmarshalJson(r io.Reader) (map[string]interface{}, error) {
	var v map[string]interface{}

	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return nil, err
	}

	return v, nil
}
