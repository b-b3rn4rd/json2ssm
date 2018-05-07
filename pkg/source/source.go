package source

import "io"

type Flattener interface {
	Flatten(io.Reader) (map[string]interface{}, error)
}
