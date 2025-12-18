package lib

import (
	"context"
	"os"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
)

const addressPrefix = "verana"

// ClientConfig holds the configuration for the Cosmos client
type ClientConfig struct {
	AddressPrefix string
	HomeDir       string
	NodeAddress   string
	Gas           string
	Fees          string
}

// DefaultConfig returns the default client configuration
func DefaultConfig() ClientConfig {
	// Get values from environment variables or use defaults
	return ClientConfig{
		AddressPrefix: getEnvOrDefault("ADDRESS_PREFIX", "verana"),
		HomeDir:       getEnvOrDefault("HOME_DIR", "~/.verana"),
		NodeAddress:   getEnvOrDefault("NODE_RPC", ""),
		Gas:           getEnvOrDefault("GAS", "300000"),
		Fees:          getEnvOrDefault("FEES", "750000uvna"),
	}
}

// NewClient creates a new Cosmos client with the given configuration
func NewClient(ctx context.Context, config ClientConfig) (cosmosclient.Client, error) {
	clientOpts := []cosmosclient.Option{
		cosmosclient.WithAddressPrefix(config.AddressPrefix),
		cosmosclient.WithHome(config.HomeDir),
		cosmosclient.WithGas(config.Gas),
		cosmosclient.WithFees(config.Fees),
	}

	if config.NodeAddress != "" {
		clientOpts = append(clientOpts, cosmosclient.WithNodeAddress(config.NodeAddress))
	}

	return cosmosclient.New(ctx, clientOpts...)
}

// getEnvOrDefault gets an environment variable or returns the default
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
