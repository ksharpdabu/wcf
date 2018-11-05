package mix_layer

type EnDec interface {
	Encode(input []byte, output []byte) (int, error)
	Decode(input []byte, output []byte) (int, error)
	InitRead(key []byte, iv []byte) error
	InitWrite(key []byte, iv []byte) error
	Name() string
	IVLen() int
}
