package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCipherRoundTrip(t *testing.T) {
	dir := t.TempDir()
	key, err := LoadOrGenerateMasterKey(dir, "")
	require.NoError(t, err)
	require.Len(t, key, 32)

	// Reload should yield identical key
	key2, err := LoadOrGenerateMasterKey(dir, "")
	require.NoError(t, err)
	assert.Equal(t, key, key2)

	// Verify file permission 0600
	info, err := os.Stat(filepath.Join(dir, "master.key"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	c, err := NewCipher(key)
	require.NoError(t, err)

	enc, err := c.EncryptString("super-secret-jwt-key")
	require.NoError(t, err)
	assert.NotEmpty(t, enc)
	assert.NotEqual(t, "super-secret-jwt-key", enc)

	dec, err := c.DecryptString(enc)
	require.NoError(t, err)
	assert.Equal(t, "super-secret-jwt-key", dec)

	emptyEnc, err := c.EncryptString("")
	require.NoError(t, err)
	assert.Empty(t, emptyEnc)
}

func TestBundleEncodeDecode(t *testing.T) {
	payload := []byte(`{"hello":"world"}`)

	// Unencrypted bundle
	plainBlob, err := EncodeBundle(payload, "")
	require.NoError(t, err)
	isEnc, err := IsBundleEncrypted(plainBlob)
	require.NoError(t, err)
	assert.False(t, isEnc)
	decoded, err := DecodeBundle(plainBlob, "")
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)

	// Encrypted bundle
	encBlob, err := EncodeBundle(payload, "secret-123")
	require.NoError(t, err)
	isEnc, err = IsBundleEncrypted(encBlob)
	require.NoError(t, err)
	assert.True(t, isEnc)

	_, err = DecodeBundle(encBlob, "")
	assert.ErrorIs(t, err, ErrPassphraseRequired)

	_, err = DecodeBundle(encBlob, "wrong-pass")
	assert.Error(t, err)

	decoded, err = DecodeBundle(encBlob, "secret-123")
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("my-admin-pass")
	require.NoError(t, err)
	assert.True(t, VerifyPassword(hash, "my-admin-pass"))
	assert.False(t, VerifyPassword(hash, "wrong-pass"))
}
