package jsonbody

import (
	"io"

	"github.com/mailru/easyjson"
)

func Decode(r io.Reader, v easyjson.Unmarshaler) error {
	return easyjson.UnmarshalFromReader(r, v)
}

func Unmarshal(data []byte, v easyjson.Unmarshaler) error {
	return easyjson.Unmarshal(data, v)
}
