package models

import "encoding/gob"

type IAmMaster struct{}

type OK struct{}

type AddMe struct{}

func init() {
	gob.Register(IAmMaster{})
	gob.Register(OK{})
	gob.Register(AddMe{})
}
