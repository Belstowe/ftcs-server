package main

import (
	"os"
	"time"

	"github.com/Belstowe/ftcs-server/statefulserver"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	var server *statefulserver.Server
	var err error
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).Level(zerolog.DebugLevel)
	if server, err = statefulserver.NewServer("ftcs-server-1:5001", "ftcs-server-2:5001", "ftcs-server-3:5001"); err != nil {
		log.Fatal().Msg(err.Error())
	}
	for {
		if err = server.Listen(); err != nil {
			log.Error().Msg(err.Error())
		}
	}
}
