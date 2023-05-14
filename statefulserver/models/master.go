package models

import "encoding/gob"

type AreYouMaster struct{}

type MasterYes struct{}

type MasterNo struct{}

func init() {
	gob.Register(AreYouMaster{})
	gob.Register(MasterYes{})
	gob.Register(MasterNo{})
}
