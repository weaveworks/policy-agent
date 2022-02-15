package gateway

import (
	"time"
)

func (g *Gateway) Send(kind PacketKind, message interface{}, response interface{}) error {
	return g.send(kind, message, response)
}

func (g *Gateway) send(kind PacketKind, message interface{}, response interface{}) error {
	var (
		req []byte
		err error
	)

	req, err = EncodeSnappy(message)
	if err != nil {
		return err
	}

	res, err := g.client.Send(kind.String(), req)
	if err != nil {
		return err
	}

	if response == nil {
		return nil
	}

	return DecodeSnappy(res, response)
}

func (g *Gateway) hello() error {
	message := PacketHello{
		Major:     ProtocolMajorVersion,
		Minor:     ProtocolMinorVersion,
		AccountID: g.accountID,
		ClusterID: g.clusterID,
	}

	err := g.send(PacketKindHello, message, nil)
	if err != nil {
		return err
	}
	return nil
}

func (g *Gateway) ping() error {
	var pong PacketPong
	err := g.send(PacketKindPing, PacketPing{Started: time.Now().UTC()}, &pong)
	if err != nil {
		return err
	}
	return nil
}
