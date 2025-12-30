// Go-side corollary: reproduce the server's legacy Amino sign bytes.
// Run from repo root:
//   go run ts-proto/test/scripts/amino-sign-bench-go.go
//
// This prints:
// - server sign bytes (legacy amino JSON with omitempty semantics)
// - client-style bytes (manually injected zero fee fields)
// so you can see the mismatch that breaks signature verification.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	permtypes "github.com/verana-labs/verana/x/perm/types"
)

func main() {
	amino := codec.NewLegacyAmino()
	permtypes.RegisterLegacyAminoCodec(amino)
	legacytx.RegressionTestingAminoCodec = amino

	effectiveFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	effectiveUntil := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	address := "verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4"
	msg := &permtypes.MsgCreatePermission{
		Creator:          address,
		SchemaId:         1,
		Type:             permtypes.PermissionType_VERIFIER,
		Did:              "did:verana:test:bench",
		Country:          "US",
		EffectiveFrom:    &effectiveFrom,
		EffectiveUntil:   &effectiveUntil,
		VerificationFees: 0,
		ValidationFees:   0,
	}

	fee := legacytx.StdFee{
		Amount: sdk.NewCoins(sdk.NewInt64Coin("uvna", 557532)),
		Gas:    185844,
	}

	accountNumber := uint64(0)
	sequence := uint64(111)
	lcdEndpoint := strings.TrimSuffix(getEnv("VERANA_LCD_ENDPOINT", "http://localhost:1317"), "/")

	// Try to query the chain for the real account number/sequence via LCD.
	// Falls back to defaults if the node is unavailable.
	if accNum, seq, ok, detail := fetchAccountNumbers(lcdEndpoint, address); ok {
		accountNumber = accNum
		sequence = seq
		fmt.Printf(
			"Connected to %s. Using on-chain account_number=%d, sequence=%d.\n",
			lcdEndpoint,
			accountNumber,
			sequence,
		)
	} else {
		fmt.Printf("Could not query %s. Using defaults account_number=0, sequence=111.\n", lcdEndpoint)
		if detail != "" {
			fmt.Printf("LCD query detail: %s\n", detail)
		}
	}

	serverBytes := legacytx.StdSignBytes(
		"vna-testnet-1",
		accountNumber,
		sequence,
		0,
		fee,
		[]sdk.Msg{msg},
		"Amino bench demo",
	)

	clientBytes := buildClientSignBytes(address, accountNumber, sequence)

	fmt.Println("Server sign bytes (legacy Amino JSON, omitempty):")
	serverJSON := string(serverBytes)
	if pretty, err := prettyJSON(serverBytes); err == nil {
		serverJSON = pretty
		fmt.Println(pretty)
	} else {
		fmt.Println(string(serverBytes))
	}
	fmt.Println("Sign bytes hex (server-style, zeros omitted):")
	fmt.Println()
	fmt.Println(hex.EncodeToString(serverBytes))
	fmt.Println()

	fmt.Println("Client-style bytes (zeros included, different bytes):")
	clientJSON := string(clientBytes)
	if pretty, err := prettyJSON(clientBytes); err == nil {
		clientJSON = pretty
		fmt.Println(pretty)
	} else {
		fmt.Println(string(clientBytes))
	}
	fmt.Println("Sign bytes hex (client-style, zeros included):")
	fmt.Println()
	fmt.Println(hex.EncodeToString(clientBytes))
	fmt.Println()

	fmt.Println("Equal?", string(serverBytes) == string(clientBytes))

	outDir := filepath.Join("ts-proto", "test", "out")
	_ = os.MkdirAll(outDir, 0o755)
	_ = os.WriteFile(filepath.Join(outDir, "amino-sign-bench-go-server.json"), []byte(serverJSON+"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(outDir, "amino-sign-bench-go-client.json"), []byte(clientJSON+"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(outDir, "amino-sign-bench-go-server.hex"), []byte(hex.EncodeToString(serverBytes)+"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(outDir, "amino-sign-bench-go-client.hex"), []byte(hex.EncodeToString(clientBytes)+"\n"), 0o644)
	fmt.Printf("Wrote Go outputs to %s\n", outDir)
}

func prettyJSON(raw []byte) (string, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(pretty), nil
}

func buildClientSignBytes(address string, accountNumber, sequence uint64) []byte {
	signDoc := map[string]any{
		"chain_id":       "vna-testnet-1",
		"account_number": fmt.Sprintf("%d", accountNumber),
		"sequence":       fmt.Sprintf("%d", sequence),
		"fee": map[string]any{
			"amount": []map[string]string{
				{
					"amount": "557532",
					"denom":  "uvna",
				},
			},
			"gas": "185844",
		},
		"msgs": []map[string]any{
			{
				"type": "/perm/v1/create-perm",
				"value": map[string]any{
					"creator":           address,
					"schema_id":         "1",
					"type":              2,
					"did":               "did:verana:test:bench",
					"country":           "US",
					"effective_from":    "2025-01-01T00:00:00Z",
					"effective_until":   "2025-12-31T00:00:00Z",
					"verification_fees": "0",
					"validation_fees":   "0",
				},
			},
		},
		"memo": "Amino bench demo",
	}

	raw, err := json.Marshal(signDoc)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal client sign bytes: %v", err))
	}

	return sdk.MustSortJSON(raw)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func fetchAccountNumbers(lcdEndpoint, address string) (uint64, uint64, bool, string) {
	url := fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", lcdEndpoint, address)
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, false, fmt.Sprintf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, 0, false, fmt.Sprintf("GET %s returned status %s", url, resp.Status)
	}

	var payload struct {
		Account struct {
			AccountNumber string `json:"account_number"`
			Sequence      string `json:"sequence"`
		} `json:"account"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, 0, false, fmt.Sprintf("GET %s decode failed: %v", url, err)
	}

	accountNumber, err1 := strconv.ParseUint(payload.Account.AccountNumber, 10, 64)
	sequence, err2 := strconv.ParseUint(payload.Account.Sequence, 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false, fmt.Sprintf(
			"GET %s parse failed: account_number=%q sequence=%q",
			url,
			payload.Account.AccountNumber,
			payload.Account.Sequence,
		)
	}
	return accountNumber, sequence, true, ""
}
