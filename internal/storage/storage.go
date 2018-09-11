//go:generate mockgen -source=storage.go -destination=storage_mock.go -package=storage Storage
package storage

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/arjantop/pwned-passwords/internal/tracing"
	"go.opencensus.io/trace"
)

// Backend is an interface representing the ability to get a io.ReadCloser for a
// requested key.
//
// A Backend is used by Storage to request underlying data.
type Backend interface {
	// Read returns a io.ReadCloser for the requested key.
	Read(ctx context.Context, key string) io.ReadCloser
}

type Storage interface {
	Get(ctx context.Context, key string) (result [][]byte, err error)
}

// ObjectStorage provides access to hashes based on a key from a Backend.
type ObjectStorage struct {
	Backend Backend
}

// Get return a list of hashes.
func (s *ObjectStorage) Get(ctx context.Context, key string) (result [][]byte, err error) {
	ctx, span := trace.StartSpan(ctx, "ObjectStorage.Get")
	defer tracing.EndSpan(span, &err)

	r := s.Backend.Read(ctx, key)
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	numHashes := len(buf) / 20
	hashes := make([][]byte, 0, numHashes)

	for i := 0; i < numHashes; i++ {
		hashes = append(hashes, buf[i*20:(i+1)*20])
	}

	return hashes, err
}
