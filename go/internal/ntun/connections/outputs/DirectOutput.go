package outputs

import "net"

type DirectOutput struct {
}

func NewDirectOutput() *DirectOutput {
	return &DirectOutput{}
}

func (d *DirectOutput) Dial(dstAddress string) (net.Conn, error) {
	return net.Dial("tcp", dstAddress)
}

func (d *DirectOutput) Listen() error {
	return nil
}

func (d *DirectOutput) Close() error {
	return nil
}
