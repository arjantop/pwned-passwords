package storage

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/arjantop/pwned-passwords/internal/tracing"
	"go.opencensus.io/trace"
)

type Backend interface {
	Read(ctx context.Context, key string) io.ReadCloser
}

type Storage interface {
	Get(ctx context.Context, key string) ([][]byte, error)
}

type RealStorage struct {
	Backend Backend
}

func (s *RealStorage) Get(ctx context.Context, key string) (result [][]byte, err error) {
	ctx, span := trace.StartSpan(ctx, "RealStorage.Get")
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
