package argon2id

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("argon2id : hash is not in the correct format")
	ErrIncompatibleVariant = errors.New("argon2id : incompatible variant of argon2")
	ErrIncompatibleVersion = errors.New("argon2id : incompatible version of argon2")
)

type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var DefaultParams = &Params{
	Memory:      64 * 1024,
	Iterations:  1,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

func generateRandByte(n uint32) ([]byte, error) {
	byte := make([]byte, n)
	_, err := rand.Read(byte)
	if err != nil {
		return nil, err
	}
	return byte, nil
}

func CreateHash(password string, params *Params) (hash string, err error) {
	salt, err := generateRandByte(params.SaltLength)
	if err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	base64salt := base64.RawStdEncoding.EncodeToString(salt)
	base64key := base64.RawStdEncoding.EncodeToString(key)
	hash = fmt.Sprintf("$argon2id$ver=%d$memo=%d,it=%d,pll=%d$%s$%s", argon2.Version, params.Memory, params.Iterations, params.Parallelism, base64salt, base64key)
	return hash, nil
}

func ComparePassAndHash(password, hash string) (match bool, err error) {
	match, _, err = CheckHash(password, hash)
	return match, err
}

func CheckHash(password, hash string) (match bool, params *Params, err error) {
	params, salt, key, err := DecodeHash(hash)
	if err != nil {
		return false, nil, err
	}

	otherKey := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)

	keyLen := int32(len(key))
	otherKeyLen := int32(len(otherKey))

	if subtle.ConstantTimeEq(keyLen, otherKeyLen) == 0 {
		return false, params, nil
	}
	if subtle.ConstantTimeCompare(key, otherKey) == 1 {
		return true, params, nil
	}
	return false, params, nil
}

func DecodeHash(hash string) (params *Params, salt, key []byte, err error) {
	vals := strings.Split(hash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	if vals[1] != "argon2id" {
		return nil, nil, nil, ErrIncompatibleVariant
	}

	var version int
	_, err = fmt.Sscanf(vals[2], "ver=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	params = &Params{}
	_, err = fmt.Sscanf(vals[3], "memo=%d,it=%d,pll=%d", &params.Memory, &params.Iterations, &params.Parallelism)
	if err != nil {
		return nil, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}
	params.SaltLength = uint32(len(salt))

	key, err = base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}
	params.KeyLength = uint32(len(key))

	return params, salt, key, nil
}