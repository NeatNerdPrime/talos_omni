// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package user

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/siderolabs/omni/client/pkg/client"
	"github.com/siderolabs/omni/client/pkg/omnictl/internal/access"
)

// deleteCmd represents the user delete command.
var deleteCmd = &cobra.Command{
	Use:     "delete [email1 email2]",
	Short:   "Delete users.",
	Long:    `Delete users with the specified emails.`,
	Example: "",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return access.WithClient(deleteUsers(args...))
	},
}

func deleteUsers(emails ...string) func(ctx context.Context, client *client.Client) error {
	return func(ctx context.Context, client *client.Client) error {
		for _, email := range emails {
			if err := client.Management().DestroyUser(ctx, email); err != nil {
				return err
			}

			fmt.Printf("destroyed user %s\n", email)
		}

		return nil
	}
}

func init() {
	userCmd.AddCommand(deleteCmd)
}
