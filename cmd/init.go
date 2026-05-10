package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/micahlee/pco-cli/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure PCO API credentials",
	Long:  "Prompts for Planning Center API credentials and stores them in the macOS Keychain.",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		clientID, err := promptLine(reader, "PCO client ID: ")
		if err != nil {
			return err
		}
		if clientID == "" {
			return fmt.Errorf("client ID cannot be empty")
		}

		secret, err := promptSecret(reader, "PCO secret: ")
		if err != nil {
			return err
		}
		if secret == "" {
			return fmt.Errorf("secret cannot be empty")
		}

		if err := config.SaveCredentialToKeychain(config.KeychainAccountClientID, clientID); err != nil {
			return err
		}
		if err := config.SaveCredentialToKeychain(config.KeychainAccountClientSecret, secret); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "PCO credentials saved to macOS Keychain.")
		return nil
	},
}

func promptLine(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptSecret(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if isTerminal(os.Stdin) {
		if err := setEcho(false); err != nil {
			return "", err
		}
		defer setEcho(true)
		defer fmt.Fprintln(os.Stderr)
	}

	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func setEcho(enabled bool) error {
	arg := "-echo"
	if enabled {
		arg = "echo"
	}
	cmd := exec.Command("stty", arg)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	rootCmd.AddCommand(initCmd)
}
