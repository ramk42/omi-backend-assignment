package natsclient

import (
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func New(url string) (*nats.Conn, error) {
	nc, err := nats.Connect(url, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Info().Msg("nats reconnected")
	}), nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		log.Error().Err(err).Msg("nats disconnected")
	}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Error().Err(nc.LastError()).Msgf("nats Connection closed")
		}),
	)

	if err != nil {
		return nil, err
	}
	log.Info().Msg("connected to nats")
	return nc, err
}
