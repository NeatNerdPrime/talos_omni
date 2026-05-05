// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

// Package logging contains zap logging helpers.
package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Component returns the well-known "component" zap field.
func Component(name string) zap.Field {
	return zap.String("component", name)
}

// IncreaseLevel raises the logger's minimum level to lvl.
//
// Unlike zap.IncreaseLevel, it is a no-op when the underlying core is already
// at or above lvl, instead of failing and printing an error to stderr.
func IncreaseLevel(logger *zap.Logger, lvl zapcore.Level) *zap.Logger {
	if logger.Level() >= lvl {
		return logger
	}

	return logger.WithOptions(zap.IncreaseLevel(lvl))
}
