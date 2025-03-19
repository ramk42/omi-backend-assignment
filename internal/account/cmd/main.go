package main

import (
	"context"
	"github.com/ramk42/omi-backend-assignment/internal/account/api/server"
	"github.com/ramk42/omi-backend-assignment/pkg/env"
	"github.com/rs/zerolog/log"

	"github.com/ramk42/omi-backend-assignment/pkg/logsreporting"
	"github.com/ramk42/omi-backend-assignment/pkg/natsclient"
	"github.com/rs/zerolog"
	"os"
)

func main() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	ctx := context.Background()
	natURL := env.MustGetEnv[string]("NATS_URL")
	conn, err := natsclient.New(natURL)
	if err != nil {
		log.Fatal().Err(err).Str("nats_url", natURL).Msg("failed to connect to nats")
		return
	}
	auditReport := logsreporting.NewProducer(conn)
	server.Run(ctx, auditReport)
}
