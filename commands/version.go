package commands

import (
	"github.com/sachamorard/swapper/response"
)

const (
	version    = "1.0.3"
)

func Version() response.Response {
	return response.Success(version)
}

