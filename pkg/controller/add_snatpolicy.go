package controller

import (
	"github.com/noironetworks/snat-operator/pkg/controller/snatpolicy"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, snatpolicy.Add)
}
