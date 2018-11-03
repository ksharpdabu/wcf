package aes_layer

type EnDec interface {
	Encode(src []byte) ([]byte, error)
	Decode(dst []byte) ([]byte, error)
	Init(key []byte, iv []byte) error
	Name() string
	Close() error
}
