package uuid

import (
	"encoding"

	"github.com/globalsign/mgo/bson"
	satori "github.com/satori/go.uuid"
)

const (
	Size = satori.Size
)

type UUID satori.UUID

var (
	Nil = UUID{}
)

var (
	_ bson.Getter              = (*UUID)(nil)
	_ bson.Setter              = (*UUID)(nil)
	_ encoding.TextMarshaler   = (*UUID)(nil)
	_ encoding.TextUnmarshaler = (*UUID)(nil)
)

func FromString(raw string) (UUID, error) {
	id, err := satori.FromString(raw)
	return UUID(id), err
}

func FromStringSlice(raws []string) ([]UUID, error) {
	uuids := make([]UUID, 0, len(raws))

	for _, raw := range raws {
		uid, err := FromString(raw)
		if err != nil {
			return nil, err
		}
		uuids = append(uuids, uid)
	}
	return uuids, nil
}

func FromBytes(raw []byte) (UUID, error) {
	id, err := satori.FromBytes(raw)
	return UUID(id), err
}

func (uuid UUID) String() string {
	return satori.UUID(uuid).String()
}

func (uuid UUID) Bytes() []byte {
	return satori.UUID(uuid).Bytes()
}

// NewV1 returns UUID based on current timestamp and MAC address.
func NewV1() UUID {
	id := satori.Must(satori.NewV1())
	return UUID(id)
}

// NewV2 returns DCE Security UUID based on POSIX UID/GID.
func NewV2(domain byte) UUID {
	id := satori.Must(satori.NewV2(domain))
	return UUID(id)
}

// NewV3 returns UUID based on MD5 hash of namespace UUID and name.
func NewV3(ns UUID, name string) UUID {
	id := satori.NewV3(satori.UUID(ns), name)
	return UUID(id)
}

// NewV4 returns random generated UUID.
func NewV4() UUID {
	id := satori.Must(satori.NewV4())
	return UUID(id)
}

// NewV5 returns UUID based on SHA-1 hash of namespace UUID and name.
func NewV5(ns UUID, name string) UUID {
	id := satori.NewV5(satori.UUID(ns), name)
	return UUID(id)
}

func (uuid UUID) GetBSON() (interface{}, error) {
	return uuid.String(), nil
}

func IsNil(id UUID) bool {
	return id == Nil
}

func (uuid UUID) IsNil() bool {
	return IsNil(uuid)
}

func (uuid *UUID) SetBSON(raw bson.Raw) error {
	var str string
	err := raw.Unmarshal(&str)
	if err != nil {
		return err
	}

	if str == "" {
		*uuid = Nil
		return nil
	}

	id, err := FromString(str)
	if err != nil {
		return err
	}

	*uuid = id

	return nil
}

func (uuid UUID) MarshalText() ([]byte, error) {
	return satori.UUID(uuid).MarshalText()
}

func (uuid *UUID) UnmarshalText(data []byte) error {
	id := satori.UUID(*uuid)
	err := id.UnmarshalText(data)
	if err != nil {
		return err
	}

	*uuid = UUID(id)
	return nil
}
