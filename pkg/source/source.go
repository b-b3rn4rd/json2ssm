package source

import "io"

type SourceFile interface {
	Flatten(io.Reader) (map[string]string, error)
}
