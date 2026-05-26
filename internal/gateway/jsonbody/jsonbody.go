package jsonbody

import (
	"io"

	"github.com/mailru/easyjson"
)

func Decode(r io.Reader, v easyjson.Unmarshaler) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return easyjson.Unmarshal(body, v)
}

func Unmarshal(data []byte, v easyjson.Unmarshaler) error {
	return easyjson.Unmarshal(data, v)
}
