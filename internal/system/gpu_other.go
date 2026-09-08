//go:build !linux

package system

import "context"

func gpuModels(context.Context) (models []string, igpu string) { return nil, "" }
