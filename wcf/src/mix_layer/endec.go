package mix_layer

type EnDec interface {
	Encode(input []byte) ([]byte, error)
	Decode(input []byte) ([]byte, error)
	InitRead(key []byte, iv []byte) error
	InitWrite(key []byte, iv []byte) error
	Name() string
	IVLen() int
}
