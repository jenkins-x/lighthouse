//go:build modhack
// +build modhack

package util

// Necessary for safely adding multi-module repo. See: https://github.com/golang/go/wiki/Modules#is-it-possible-to-add-a-module-to-a-multi-module-repository
import _ "github.com/Azure/go-autorest"

// This file, and the github.com/Azure/go-autorest import, won't actually become part of  the resultant binary.
