package blockchain

type TxOutput struct {
	Value    int
	PubKey   string
	Metadata []byte
}

type TxInput struct {
	ID        []byte
	OutIndex  int
	Signature string
	StateProof []byte
}

func (in *TxInput) CanUnlock(data string) bool {
	return in.Signature == data
}

func (out *TxOutput) CanBeUnlocked(data string) bool {
	return out.PubKey == data
}