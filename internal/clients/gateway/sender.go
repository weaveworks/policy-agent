package gateway

import (
	"time"

	"github.com/MagalixTechnologies/core/packet"
)

// Send sends a message to the Saas gateway
func (g *Gateway) Send(kind packet.PacketKind, message interface{}, response interface{}) error {
	var (
		req []byte
		err error
	)

	req, err = encodeSnappy(message)
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

	return decodeSnappy(res, response)
}

// hello sends a hello packet at the start of a connection containing necessary information for initialization
func (g *Gateway) hello() error {
	message := packet.PacketHello{
		Major:            packet.ProtocolMajorVersion,
		Minor:            packet.ProtocolMinorVersion,
		AccountID:        g.accountID,
		ClusterID:        g.clusterID,
		K8sServerVersion: g.k8sServerVersion,
		ClusterProvider:  g.clusterProvider,
		AgentPermissions: g.agentPermissions,
		BuildVersion:     g.buildVersion,
	}

	err := g.Send(packet.PacketKindHello, message, nil)
	if err != nil {
		return err
	}
	return nil
}

// ping sends a ping to the Saas gateway
func (g *Gateway) ping() error {
	var pong packet.PacketPong
	err := g.Send(packet.PacketKindPing, packet.PacketPing{Started: time.Now().UTC()}, &pong)
	if err != nil {
		return err
	}
	return nil
}
