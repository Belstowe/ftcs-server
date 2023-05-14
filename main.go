package main

import (
	"os"
	"time"

	"github.com/Belstowe/ftcs-server/statefulserver"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).Level(zerolog.InfoLevel)
	if _, err := statefulserver.NewServer("ftcs-server-1:5001", "ftcs-server-2:5001", "ftcs-server-3:5001"); err != nil {
		log.Fatal().Msg(err.Error())
	}
	log.Info().Msg("listening on 0.0.0.0:5000...")
}
