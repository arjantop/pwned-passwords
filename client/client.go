package client

import (
	"context"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"io"

	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	"github.com/pkg/errors"
)

type Client struct {
	C pwnedpasswords.PwnedPasswordsClient
}

func (c *Client) IsPasswordPwned(ctx context.Context, password string) (bool, error) {
	hash := sha1.Sum([]byte(password))
	// Take first five hex characters from the computed hash
	prefix := hex.EncodeToString(hash[:3])[:5]

	r, err := c.C.ListHashesForPrefix(ctx, &pwnedpasswords.ListRequest{
		HashPrefix: prefix,
	})
	if err != nil {
		return false, errors.WithMessage(err, "call failed")
	}

	// Always receive and compare all hashes so we do not leak any timing information to the server
	// by closing the connection early.
	var matchFound bool
	for {
		h, err := r.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, errors.WithMessage(err, "receive failed")
		}
		if subtle.ConstantTimeCompare(hash[:], h.Hash) == 1 {
			matchFound = true
		}
	}

	return matchFound, nil
}
