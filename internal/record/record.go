package record

import (
	"encoding/binary"
)

const NameSize = 32

type Record struct {
	ID   uint32
	Age  uint32
	Name [NameSize]byte
}

const RecordSize = 4 + 4 + NameSize

func (r *Record) Serialize() [RecordSize]byte {
	var data [RecordSize]byte

	binary.LittleEndian.PutUint32(data[0:4], r.ID)
	binary.LittleEndian.PutUint32(data[4:8], r.Age)

	copy(data[8:], r.Name[:])

	return data
}

func Deserialize(data [RecordSize]byte) Record {
	var r Record

	r.ID = binary.LittleEndian.Uint32(data[0:4])
	r.Age = binary.LittleEndian.Uint32(data[4:8])

	copy(r.Name[:], data[8:])

	return r
}