// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/omni/internal/pkg/auth"
	"github.com/siderolabs/omni/internal/pkg/ctxstore"
)

func TestIdentityFromContext(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		ctx  context.Context //nolint:containedctx // fixture
		want string
	}{
		{
			name: "empty context",
			ctx:  context.Background(),
			want: "",
		},
		{
			name: "identity set",
			ctx:  ctxstore.WithValue(context.Background(), auth.IdentityContextKey{Identity: "alice"}),
			want: "alice",
		},
		{
			name: "verified email set",
			ctx:  ctxstore.WithValue(context.Background(), auth.VerifiedEmailContextKey{Email: "alice@example.com"}),
			want: "alice@example.com",
		},
		{
			name: "identity wins over verified email",
			ctx: ctxstore.WithValue(
				ctxstore.WithValue(context.Background(), auth.VerifiedEmailContextKey{Email: "alice@example.com"}),
				auth.IdentityContextKey{Identity: "service-account"},
			),
			want: "service-account",
		},
		{
			name: "empty identity falls back to verified email",
			ctx: ctxstore.WithValue(
				ctxstore.WithValue(context.Background(), auth.VerifiedEmailContextKey{Email: "alice@example.com"}),
				auth.IdentityContextKey{Identity: ""},
			),
			want: "alice@example.com",
		},
		{
			name: "both keys present but empty",
			ctx: ctxstore.WithValue(
				ctxstore.WithValue(context.Background(), auth.VerifiedEmailContextKey{Email: ""}),
				auth.IdentityContextKey{Identity: ""},
			),
			want: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, auth.IdentityFromContext(tc.ctx))
		})
	}
}
