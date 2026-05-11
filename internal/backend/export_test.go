// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package backend

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/siderolabs/omni/internal/backend/imagefactory"
)

func MakeTalosctlHandler(imageFactoryClient *imagefactory.Client, logger *zap.Logger) (http.Handler, error) {
	return makeTalosctlHandler(imageFactoryClient, logger)
}
