package source_test

import (
	"io"
	"testing"

	"os"

	"github.com/b-b3rn4rd/json2ssm/pkg/source"
	"github.com/stretchr/testify/assert"
)

func TestSourceJson(t *testing.T) {
	tests := map[string]struct {
		r        io.Reader
		response map[string]interface{}
		err      error
	}{
		"simplemap": {
			r: func() io.Reader { r, _ := os.Open("testdata/simplemap.json"); return r }(),
			response: map[string]interface{}{
				"name":                   "bernard",
				"address/city":           "melbourne",
				"address/code":           float64(3000),
				"address/address/street": "flinders",
				"address/address/number": float64(1),
			},
		},
		"simpleslice": {
			r: func() io.Reader { r, _ := os.Open("testdata/simpleslice.json"); return r }(),
			response: map[string]interface{}{
				"0/name": "bernard",
				"1/name": "keith",
			},
		},
		"mapinslice": {
			r: func() io.Reader { r, _ := os.Open("testdata/mapinslice.json"); return r }(),
			response: map[string]interface{}{
				"0/name":     "bernard",
				"0/colors/0": "red",
				"0/colors/1": "blue",
				"1/name":     "keith",
				"1/colors/0": "black",
				"1/colors/1": "white",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s := source.JSON{}
			r, err := s.Flatten(test.r)
			assert.Equal(t, test.response, r)
			if err != nil {
				assert.Error(t, err, test.err)
			}
		})
	}
}
