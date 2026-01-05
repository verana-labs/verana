// Go-side corollary: reproduce legacy Amino sign bytes for MsgCreateCredentialSchema.
// Run from repo root:
//   go run ts-proto/test/scripts/benches/amino/cs/go.go
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

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	cstypes "github.com/verana-labs/verana/x/cs/types"
)

const jsonSchema = `{"$id":"vpr:verana:VPR_CHAIN_ID/cs/v1/js/VPR_CREDENTIAL_SCHEMA_ID","$schema":"https://json-schema.org/draft/2020-12/schema","title":"ExampleCredential","description":"ExampleCredential using JsonSchema","type":"object","properties":{"credentialSubject":{"type":"object","properties":{"id":{"type":"string","format":"uri"},"firstName":{"type":"string","minLength":0,"maxLength":256},"lastName":{"type":"string","minLength":1,"maxLength":256},"expirationDate":{"type":"string","format":"date"},"countryOfResidence":{"type":"string","minLength":2,"maxLength":2}},"required":["id","lastName","expirationDate","countryOfResidence"]}}}`

func main() {
	amino := codec.NewLegacyAmino()
	cstypes.RegisterLegacyAminoCodec(amino)
	legacytx.RegressionTestingAminoCodec = amino

	address := "verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4"
	msg := &cstypes.MsgCreateCredentialSchema{
		Creator: address,
		TrId:    1,
		JsonSchema: jsonSchema,
		IssuerGrantorValidationValidityPeriod:   &cstypes.OptionalUInt32{Value: 0},
		VerifierGrantorValidationValidityPeriod: &cstypes.OptionalUInt32{Value: 0},
		IssuerValidationValidityPeriod:          &cstypes.OptionalUInt32{Value: 0},
		VerifierValidationValidityPeriod:        &cstypes.OptionalUInt32{Value: 180},
		HolderValidationValidityPeriod:          &cstypes.OptionalUInt32{Value: 0},
		IssuerPermManagementMode:                2,
		VerifierPermManagementMode:              1,
	}

	fee := legacytx.StdFee{
		Amount: sdk.NewCoins(sdk.NewInt64Coin("uvna", 557532)),
		Gas:    185844,
	}

	accountNumber := uint64(0)
	sequence := uint64(111)
	lcdEndpoint := strings.TrimSuffix(getEnv("VERANA_LCD_ENDPOINT", "http://localhost:1317"), "/")

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

	fmt.Println("Server sign bytes (legacy Amino JSON):")
	serverJSON := string(serverBytes)
	if pretty, err := prettyJSON(serverBytes); err == nil {
		serverJSON = pretty
		fmt.Println(pretty)
	} else {
		fmt.Println(string(serverBytes))
	}
	fmt.Println("Sign bytes hex:")
	fmt.Println()
	fmt.Println(hex.EncodeToString(serverBytes))
	fmt.Println()

	outDir := filepath.Join("ts-proto", "test", "out", "amino", "cs")
	_ = os.MkdirAll(outDir, 0o755)
	_ = os.WriteFile(filepath.Join(outDir, "amino-sign-bench-cs-go.json"), []byte(serverJSON+"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(outDir, "amino-sign-bench-cs-go.hex"), []byte(hex.EncodeToString(serverBytes)+"\n"), 0o644)
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
