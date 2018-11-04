package mix_layer

type EnDec interface {
	Encode(input []byte, output []byte) (int, error)
	Decode(input []byte, output []byte) (int, error)
	Init(key []byte, iv []byte) error
	Name() string
}
