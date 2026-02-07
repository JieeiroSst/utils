package codec

import (
	"encoding/json"

	"github.com/JIeeiroSst/utils/gosocket/core"
)

type JSONCodec struct{}

func NewJSONCodec() *JSONCodec {
	return &JSONCodec{}
}

func (c *JSONCodec) Encode(msg *core.Message) ([]byte, error) {
	return json.Marshal(msg)
}

func (c *JSONCodec) Decode(data []byte) (*core.Message, error) {
	var msg core.Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *JSONCodec) Name() string {
	return "json"
}
