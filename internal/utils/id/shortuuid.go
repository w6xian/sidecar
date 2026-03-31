package id

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/lithammer/shortuuid/v4"
	"github.com/oklog/ulid/v2"
)

type base58Encoder struct{}

func (enc base58Encoder) Encode(u uuid.UUID) string {
	return base58.Encode(u[:])
}

func (enc base58Encoder) Decode(s string) (uuid.UUID, error) {
	return uuid.FromBytes(base58.Decode(s))
}
func ShortID() string {
	enc := base58Encoder{}
	return shortuuid.NewWithEncoder(enc)
}

func ShortULID() string {
	return ulid.Make().String()
}

func DeviceId(appKey string) string {
	m, err := machineid.ProtectedID(appKey)
	if err != nil {
		m = "notfound!!!"
	}
	return m
}

// 生成uuid
func GetUuid() string {
	return uuid.NewString()
}
