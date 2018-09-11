package main

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/arjantop/pwned-passwords/internal/grpctest"
	"github.com/arjantop/pwned-passwords/internal/storage"
	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func createService(storage storage.Storage) (pwnedpasswords.PwnedPasswordsClient, *grpctest.Server) {
	s := &server{
		storage: storage,
	}

	testServer := grpctest.NewServer(func(srv *grpc.Server) {
		pwnedpasswords.RegisterPwnedPasswordsServer(srv, s)
	})

	c := pwnedpasswords.NewPwnedPasswordsClient(testServer.ClientConn())

	return c, testServer
}

func TestServerListHashesForPrefixReturnsHashes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := storage.NewMockStorage(ctrl)
	mockStorage.EXPECT().Get(gomock.Any(), "aaaaa").Return([][]byte{
		[]byte("abcdef"),
		[]byte("123456"),
	}, nil)

	c, s := createService(mockStorage)
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.ListHashesForPrefix(ctx, &pwnedpasswords.ListRequest{
		HashPrefix: "aaaaa",
	})
	var hashes [][]byte
	if assert.NoError(t, err) {
		for {
			r, err := resp.Recv()
			if err == io.EOF {
				break
			}
			assert.NoError(t, err)
			hashes = append(hashes, r.Hash)
		}
	}

	assert.Equal(t, [][]byte{
		[]byte("abcdef"),
		[]byte("123456"),
	}, hashes)
}

type errorStorage struct {
	err error
}

func (s *errorStorage) Get(ctx context.Context, key string) (result [][]byte, err error) {
	return nil, s.err
}

func TestServerListHashesForPrefixFailsWithGenericError(t *testing.T) {
	c, s := createService(&errorStorage{errors.New("my error")})
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.ListHashesForPrefix(ctx, &pwnedpasswords.ListRequest{
		HashPrefix: "aaaaa",
	})

	if assert.NoError(t, err) {
		_, err := resp.Recv()
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "my error")
	}
}

func TestServerListHashesForPrefixFailsIfHashPrefixIsOfInvalidLength(t *testing.T) {
	c, s := createService(nil)
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.ListHashesForPrefix(ctx, &pwnedpasswords.ListRequest{
		HashPrefix: "aa",
	})

	if assert.NoError(t, err) {
		_, err := resp.Recv()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prefix length must be")
	}
}
