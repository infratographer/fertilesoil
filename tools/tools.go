//go:build tools
// +build tools

// Package tools is a dummy package to force go mod to vendor the tools we use
package tools

import (
	//nolint:typecheck // This is a dummy package to force go mod to vendor the tools we use
	_ "github.com/deepmap/oapi-codegen/cmd/oapi-codegen"
)
