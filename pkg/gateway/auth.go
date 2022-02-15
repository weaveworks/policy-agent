package gateway

import (
	"crypto/sha512"
	"net/http"

	"github.com/MagalixTechnologies/channel"
	"github.com/pkg/errors"
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

func (g *Gateway) authorize() error {
	request := PacketAuthorizationRequest{
		AccountID: g.accountID,
		ClusterID: g.clusterID,
	}

	var question PacketAuthorizationQuestion
	err := g.send(PacketKindAuthorizationRequest, request, &question)
	if err != nil {
		return err
	}

	if len(question.Token) < 1024 {
		return errors.Wrapf(
			err,
			"server asks authorization/answer with unsecured token; token length: %d, token: %s",
			len(question.Token),
			string(question.Token),
		)
	}

	token, err := g.getAuthorizationToken(question.Token)
	if err != nil {
		return err
	}

	answer := PacketAuthorizationAnswer{
		Token: token,
	}

	var success PacketAuthorizationSuccess
	err = g.send(PacketKindAuthorizationAnswer, answer, &success)
	if err != nil {
		if e, ok := err.(*channel.ProtocolError); ok {
			switch e.Code {
			case http.StatusNotFound, http.StatusUnauthorized, http.StatusForbidden:
				return FatalError{Err: e}
			default:
				return e
			}
		}
		return err
	}
	return nil
}
