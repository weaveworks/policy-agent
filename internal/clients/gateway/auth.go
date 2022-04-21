package gateway

import (
	"crypto/sha512"
	"fmt"
	"net/http"

	"github.com/MagalixTechnologies/channel"
	"github.com/MagalixTechnologies/core/packet"
)

func (g *Gateway) getAuthorizationToken(question []byte) ([]byte, error) {
	payload := []byte{}
	payload = append(payload, question...)
	payload = append(payload, g.secret...)
	payload = append(payload, question...)

	sha := sha512.New()
	_, err := sha.Write(payload)
	if err != nil {
		return nil, err
	}

	return sha.Sum(nil), nil
}

// authorize authorizes against the SaaS gateway
// sends an authorization request and verififes the authorization using the returned question and the worker secret
func (g *Gateway) authorize() error {
	request := packet.PacketAuthorizationRequest{
		AccountID: g.accountID,
		ClusterID: g.clusterID,
	}

	var question packet.PacketAuthorizationQuestion
	err := g.Send(packet.PacketKindAuthorizationRequest, request, &question)
	if err != nil {
		return err
	}

	if len(question.Token) < 1024 {
		return fmt.Errorf(
			"server asks authorization/answer with unsecured token; token length: %d, err: %w",
			len(question.Token),
			err,
		)
	}

	token, err := g.getAuthorizationToken(question.Token)
	if err != nil {
		return err
	}

	answer := packet.PacketAuthorizationAnswer{
		Token: token,
	}

	var success packet.PacketAuthorizationSuccess
	err = g.Send(packet.PacketKindAuthorizationAnswer, answer, &success)
	if err != nil {
		if e, ok := err.(*channel.ProtocolError); ok {
			switch e.Code {
			case http.StatusNotFound, http.StatusUnauthorized, http.StatusForbidden:
				return ConnectionError{Err: e}
			default:
				return e
			}
		}
		return err
	}
	return nil
}
