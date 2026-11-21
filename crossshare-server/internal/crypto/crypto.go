package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"go.uber.org/fx"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"

	"crossshare-server/internal/config"
)

var Module = fx.Options(
	fx.Provide(ProvideCipher),
)

var (
	ulidMu      sync.Mutex
	ulidEntropy = ulid.Monotonic(rand.Reader, 0)
)

// NewULID generates a lexicographically sortable unique identifier.
func NewULID() string {
	ulidMu.Lock()
	defer ulidMu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), ulidEntropy).String()
}

// Cipher provides AES-256-GCM encryption for database-persisted secrets.
type Cipher struct {
	key []byte
}

// ProvideCipher loads or generates the master key and constructs a Cipher for Fx DI.
func ProvideCipher(cfg *config.Config) (*Cipher, error) {
	key, err := LoadOrGenerateMasterKey(cfg.Server.DataDir, cfg.Server.MasterKey)
	if err != nil {
		return nil, err
	}
	return NewCipher(key)
}

// LoadOrGenerateMasterKey resolves a 32-byte key from configuredKey or <dataDir>/master.key.
func LoadOrGenerateMasterKey(dataDir, configuredKey string) ([]byte, error) {
	if raw := strings.TrimSpace(configuredKey); raw != "" {
		return parseMasterKey(raw)
	}
	if strings.TrimSpace(dataDir) == "" {
		dataDir = "./data"
	}
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("create data_dir: %w", err)
	}
	keyPath := filepath.Join(dataDir, "master.key")
	if b, err := os.ReadFile(keyPath); err == nil {
		return parseMasterKey(strings.TrimSpace(string(b)))
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read master.key: %w", err)
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate master key: %w", err)
	}
	encoded := hex.EncodeToString(key) + "\n"
	if err := os.WriteFile(keyPath, []byte(encoded), 0o600); err != nil {
		return nil, fmt.Errorf("write master.key: %w", err)
	}
	return key, nil
}

func parseMasterKey(raw string) ([]byte, error) {
	if len(raw) == 64 {
		if b, err := hex.DecodeString(raw); err == nil && len(b) == 32 {
			return b, nil
		}
	}
	if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	return nil, errors.New("master_key must be a 32-byte hex (64 chars) or base64 string")
}

// NewCipher constructs a Cipher using a 32-byte master key.
func NewCipher(masterKey []byte) (*Cipher, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("expected 32-byte key, got %d", len(masterKey))
	}
	cp := make([]byte, 32)
	copy(cp, masterKey)
	return &Cipher{key: cp}, nil
}

// DeriveSubKey returns a deterministic 32-byte subkey for a specific purpose (e.g. admin JWT signing).
func (c *Cipher) DeriveSubKey(purpose string) []byte {
	mac := hmac.New(sha256.New, c.key)
	mac.Write([]byte(purpose))
	return mac.Sum(nil)
}

// EncryptString encrypts plaintext and returns a base64-encoded ciphertext.
// An empty string returns an empty string.
func (c *Cipher) EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	enc, err := encryptAESGCM(c.key, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(enc), nil
}

// DecryptString decrypts a base64-encoded ciphertext produced by EncryptString.
// An empty string returns an empty string.
func (c *Cipher) DecryptString(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode base64 ciphertext: %w", err)
	}
	plain, err := decryptAESGCM(c.key, raw)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

var (
	BundleMagicPlain     = []byte{'C', 'S', 'B', '0'}
	BundleMagicEncrypted = []byte{'C', 'S', 'B', '1'}
)

// ErrPassphraseRequired indicates the bundle is encrypted and requires a passphrase.
var ErrPassphraseRequired = errors.New("passphrase is required for encrypted migration bundle")

// IsBundleEncrypted inspects the bundle header and reports whether it is encrypted.
func IsBundleEncrypted(blob []byte) (bool, error) {
	if len(blob) < 4 {
		return false, errors.New("invalid migration bundle format")
	}
	if bytes.Equal(blob[:4], BundleMagicEncrypted) {
		return true, nil
	}
	if bytes.Equal(blob[:4], BundleMagicPlain) {
		return false, nil
	}
	return false, errors.New("unsupported migration bundle header")
}

// EncodeBundle wraps payload bytes with either plain (CSB0) or passphrase-encrypted (CSB1) framing.
func EncodeBundle(plaintext []byte, passphrase string) ([]byte, error) {
	if passphrase == "" {
		out := make([]byte, 0, len(BundleMagicPlain)+len(plaintext))
		out = append(out, BundleMagicPlain...)
		out = append(out, plaintext...)
		return out, nil
	}
	return EncryptWithPassphrase(plaintext, passphrase)
}

// DecodeBundle unwraps either plain (CSB0) or passphrase-encrypted (CSB1) bundle framing.
func DecodeBundle(blob []byte, passphrase string) ([]byte, error) {
	encrypted, err := IsBundleEncrypted(blob)
	if err != nil {
		return nil, err
	}
	if !encrypted {
		return blob[len(BundleMagicPlain):], nil
	}
	if passphrase == "" {
		return nil, ErrPassphraseRequired
	}
	return DecryptWithPassphrase(blob, passphrase)
}

// EncryptWithPassphrase encrypts arbitrary bytes using Argon2id key derivation + AES-256-GCM.
func EncryptWithPassphrase(plaintext []byte, passphrase string) ([]byte, error) {
	if len(passphrase) < 6 {
		return nil, errors.New("passphrase must be at least 6 characters")
	}
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	key := argon2.IDKey([]byte(passphrase), salt, 2, 64*1024, 4, 32)
	enc, err := encryptAESGCM(key, plaintext)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(BundleMagicEncrypted)+len(salt)+len(enc))
	out = append(out, BundleMagicEncrypted...)
	out = append(out, salt...)
	out = append(out, enc...)
	return out, nil
}

// DecryptWithPassphrase decrypts a payload produced by EncryptWithPassphrase.
func DecryptWithPassphrase(blob []byte, passphrase string) ([]byte, error) {
	headerLen := len(BundleMagicEncrypted) + 16
	if len(blob) <= headerLen {
		return nil, errors.New("invalid migration bundle format")
	}
	if !bytes.Equal(blob[:len(BundleMagicEncrypted)], BundleMagicEncrypted) {
		return nil, errors.New("unsupported migration bundle header")
	}
	salt := blob[len(BundleMagicEncrypted):headerLen]
	ciphertext := blob[headerLen:]
	key := argon2.IDKey([]byte(passphrase), salt, 2, 64*1024, 4, 32)
	plain, err := decryptAESGCM(key, ciphertext)
	if err != nil {
		return nil, errors.New("invalid passphrase or corrupted migration bundle")
	}
	return plain, nil
}

func encryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, 0, len(nonce)+len(sealed))
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, nil
}

func decryptAESGCM(key, payload []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(payload) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// HashPassword hashes an admin password with bcrypt.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword compares a bcrypt hash against a plaintext password.
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// RandomToken returns a URL-safe random hex token of n bytes.
func RandomToken(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
