package storage

import (
	"context"
	"io"
	"io/ioutil"
	"path"

	"os"

	"github.com/arjantop/pwned-passwords/internal/filename"
	"go.opencensus.io/trace"
)

type errReader struct {
	err error
}

func (r *errReader) Read(p []byte) (int, error) {
	return 0, r.err
}

type LocalBackend struct {
	Dir string
}

func (s *LocalBackend) Read(ctx context.Context, key string) io.ReadCloser {
	ctx, span := trace.StartSpan(ctx, "LocalBackend.Read")
	defer span.End()

	filePath := filename.PathFor(key, ".bin")

	f, err := os.Open(path.Join(s.Dir, filePath))
	if err != nil {
		return ioutil.NopCloser(&errReader{err})
	}

	return f
}
