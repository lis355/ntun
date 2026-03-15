package messages

type Messenger interface {
	SendMessage(buf []byte) error
	RecieveMessage() ([]byte, error)
}
