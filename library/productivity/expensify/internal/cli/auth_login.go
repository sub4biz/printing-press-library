// Copyright 2026 Matt Van Horn and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored `auth login` command. The headline path is `--from-chrome`,
// which reads the freshest www.expensify.com authToken straight out of the
// user's already-signed-in Chrome session (decrypting Chrome's cookie store)
// so no token copy/paste is needed. Recorded in .printing-press-patches.json
// as `auth-from-chrome`.

package cli

import (
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/config"

	"github.com/spf13/cobra"
)

func newAuthLoginCmd(flags *rootFlags) *cobra.Command {
	var fromChrome bool
	var token string
	var debugPort int

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Capture an Expensify session authToken (from Chrome or pasted)",
		Long: strings.TrimSpace(`
Capture the Expensify session authToken and save it to the local config.

  --from-chrome   read the freshest authToken from your already-signed-in Chrome
  --token <t>     provide the token directly (no browser)

--from-chrome decrypts Chrome's cookie store and picks the most recently updated
www.expensify.com authToken (including a not-yet-checkpointed value in the WAL
sidecar), so a fresh browser login is picked up immediately. Chrome 127+
App-Bound Encryption (v20 cookies) cannot be read this way; paste with --token
or 'auth set-token' in that case.`),
		Example: strings.Trim(`
  expensify-pp-cli auth login --from-chrome
  expensify-pp-cli auth login --token <authToken>`, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would capture an Expensify authToken and save it to config")
				return nil
			}
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			w := cmd.OutOrStdout()

			if token != "" {
				if err := cfg.SaveCredential(strings.TrimSpace(token)); err != nil {
					return configErr(err)
				}
				fmt.Fprintf(w, "Token saved to %s\n", cfg.Path)
				return nil
			}

			if !fromChrome {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("pass --from-chrome to read your Chrome session, or --token <t> to set one directly"))
			}

			tok, email, cerr := captureTokenFromChrome(debugPort)
			if cerr != nil {
				return fmt.Errorf("could not read authToken from Chrome: %w (sign in to www.expensify.com in Chrome, or use --token)", cerr)
			}
			if err := cfg.SaveCredential(tok); err != nil {
				return configErr(err)
			}
			if email != "" {
				fmt.Fprintf(w, "Captured session token (%d chars) for %s. Saved to %s\n", len(tok), email, cfg.Path)
			} else {
				fmt.Fprintf(w, "Captured session token (%d chars) from Chrome. Saved to %s\n", len(tok), cfg.Path)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fromChrome, "from-chrome", false, "Read the authToken from your already-signed-in Chrome")
	cmd.Flags().StringVar(&token, "token", "", "Provide the authToken directly instead of reading Chrome")
	cmd.Flags().IntVar(&debugPort, "chrome-debug-port", 0, "If set, read cookies from a Chrome started with --remote-debugging-port=<port>")
	return cmd
}
