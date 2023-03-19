package main

import (
	"log"

	"github.com/Belstowe/ftcs-server/statefulserver"
)

func main() {
	srv := assert(statefulserver.NewServer())
	for {
		if err := srv.PeerListen(); err != nil {
			log.Println(err)
		}
	}
}

func assert[T any](res T, err error) T {
	if err != nil {
		log.Fatalln(err)
	}
	return res
}
