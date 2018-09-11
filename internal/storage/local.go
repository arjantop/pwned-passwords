package storage

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"

	"go.opencensus.io/trace"
)

func NewLocalStorage(dir string) Storage {
	return &ObjectStorage{
		Backend: &LocalBackend{Dir: dir},
	}
}

type errReader struct {
	err error
}

func (r *errReader) Read(p []byte) (int, error) {
	return 0, r.err
}

// LocalBackend is a storage backend that reads files from the local filesystem.
type LocalBackend struct {
	Dir string
}

func (s *LocalBackend) Read(ctx context.Context, key string) io.ReadCloser {
	ctx, span := trace.StartSpan(ctx, "LocalBackend.Read")
	defer span.End()

	filePath := PathFor(key, ".bin")

	f, err := os.Open(path.Join(s.Dir, filePath))
	if err != nil {
		return ioutil.NopCloser(&errReader{err})
	}

	return f
}
