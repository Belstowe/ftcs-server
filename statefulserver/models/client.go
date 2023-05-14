package models

import "encoding/gob"

type ClientError struct {
	Message string
}

func init() {
	gob.Register(ClientError{})
}
