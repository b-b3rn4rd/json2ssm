package source

import (
	"encoding/json"
	"io"

	"io/ioutil"

	"github.com/nytlabs/gojsonexplode"
)

type JSON struct{}

func (j *JSON) Flatten(r io.Reader) (map[string]interface{}, error) {
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	jsonRaw, err := gojsonexplode.Explodejson(raw, "/")
	if err != nil {
		return nil, err
	}

	var v map[string]interface{}

	err = json.Unmarshal(jsonRaw, &v)
	if err != nil {
		return nil, err
	}

	return v, nil
}
