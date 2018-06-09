package storage

import (
	"context"
	"io/ioutil"
	"path"

	"github.com/arjantop/pwned-passwords/internal/filename"
	"go.opencensus.io/trace"
)

type Local struct {
	Dir string
}

func (s *Local) Get(ctx context.Context, key string) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "Local.Get")
	defer span.End()

	filePath := filename.PathFor(key, ".bin")

	buf, err := ioutil.ReadFile(path.Join(s.Dir, filePath))
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
