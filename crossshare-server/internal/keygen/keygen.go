package keygen

import (
	"crypto/rand"
	"math/big"
)

// Base58: removes 0, O, I, l to avoid ambiguity when reading/copying keys.
const charset = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

var base = big.NewInt(int64(len(charset)))

func Generate(length int) (string, error) {
	b := make([]byte, length)
	max := big.NewInt(int64(len(charset)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

// FromHash encodes raw hash bytes as base62 and returns the first length characters.
func FromHash(hash []byte, length int) string {
	n := new(big.Int).SetBytes(hash)
	buf := make([]byte, 0, 44)
	zero := big.NewInt(0)
	for n.Cmp(zero) > 0 {
		mod := new(big.Int)
		n.DivMod(n, base, mod)
		buf = append(buf, charset[mod.Int64()])
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	if length > len(buf) {
		length = len(buf)
	}
	return string(buf[:length])
}
