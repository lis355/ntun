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
