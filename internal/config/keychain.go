package config

import (
	"fmt"
	"runtime"
	"strings"

	"os/exec"
)

const (
	KeychainService             = "pco-cli"
	KeychainAccountClientID     = "client_id"
	KeychainAccountClientSecret = "client_secret"
)

// SaveCredentialToKeychain stores a credential in the macOS login Keychain.
func SaveCredentialToKeychain(account, value string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("keychain credential storage is only supported on macOS")
	}

	cmd := exec.Command("security", "add-generic-password", "-U", "-s", KeychainService, "-a", account, "-w", value)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("storing %s in Keychain: %w: %s", account, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// LoadCredentialFromKeychain reads a credential from the macOS login Keychain.
func LoadCredentialFromKeychain(account string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("keychain credential storage is only supported on macOS")
	}

	cmd := exec.Command("security", "find-generic-password", "-s", KeychainService, "-a", account, "-w")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("reading %s from Keychain: %w", account, err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}
