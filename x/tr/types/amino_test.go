package types_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/x/tx/signing/aminojson"

	dev1 "github.com/verana-labs/verana/api/verana/de/v1"
	trv1 "github.com/verana-labs/verana/api/verana/tr/v1"
)

func TestAminoJSONEncoder(t *testing.T) {
	enc := aminojson.NewEncoder(aminojson.EncoderOptions{})

	// MsgCreateTrustRegistry
	msg := &trv1.MsgCreateTrustRegistry{
		Authority:    "verana1f06a8j6n02ash0vtkdge0a9pvvfa5ypghkyttt",
		Operator:     "verana13627hsukut3p39l5lawa88er0az36227rkwt83",
		Did:          "did:verana:test:1234567890:abc123",
		Aka:          "http://ts-proto-test-trust-registry.com",
		Language:     "en",
		DocUrl:       "https://example.com/governance-framework.pdf",
		DocDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
	}
	bz, _ := enc.Marshal(msg)
	fmt.Printf("MsgCreateTrustRegistry value:\n%s\n\n", string(bz))

	// MsgUpdateTrustRegistry (has uint64 id)
	updateMsg := &trv1.MsgUpdateTrustRegistry{
		Authority: "verana1f06a8j6n02ash0vtkdge0a9pvvfa5ypghkyttt",
		Operator:  "verana13627hsukut3p39l5lawa88er0az36227rkwt83",
		Id:        1,
		Did:       "did:verana:test:updated:abc123",
		Aka:       "http://updated-trust-registry.com",
	}
	bz2, _ := enc.Marshal(updateMsg)
	fmt.Printf("MsgUpdateTrustRegistry value:\n%s\n\n", string(bz2))

	// MsgArchiveTrustRegistry
	archiveMsg := &trv1.MsgArchiveTrustRegistry{
		Authority: "verana1f06a8j6n02ash0vtkdge0a9pvvfa5ypghkyttt",
		Operator:  "verana13627hsukut3p39l5lawa88er0az36227rkwt83",
		Id:        1,
		Archive:   true,
	}
	bz3, _ := enc.Marshal(archiveMsg)
	fmt.Printf("MsgArchiveTrustRegistry value:\n%s\n\n", string(bz3))

	// MsgAddGovernanceFrameworkDocument
	addGfdMsg := &trv1.MsgAddGovernanceFrameworkDocument{
		Authority:    "verana1f06a8j6n02ash0vtkdge0a9pvvfa5ypghkyttt",
		Operator:     "verana13627hsukut3p39l5lawa88er0az36227rkwt83",
		Id:           1,
		DocLanguage:  "en",
		DocUrl:       "https://example.com/governance-framework-v2.pdf",
		DocDigestSri: "sha384-TsProtoTestDocHash1234567890123456789012345678901234567890123456789012345678",
		Version:      2,
	}
	bz4, _ := enc.Marshal(addGfdMsg)
	fmt.Printf("MsgAddGovernanceFrameworkDocument value:\n%s\n\n", string(bz4))

	// MsgIncreaseActiveGovernanceFrameworkVersion
	increaseMsg := &trv1.MsgIncreaseActiveGovernanceFrameworkVersion{
		Authority: "verana1f06a8j6n02ash0vtkdge0a9pvvfa5ypghkyttt",
		Operator:  "verana13627hsukut3p39l5lawa88er0az36227rkwt83",
		Id:        1,
	}
	bz5, _ := enc.Marshal(increaseMsg)
	fmt.Printf("MsgIncreaseActiveGovernanceFrameworkVersion value:\n%s\n\n", string(bz5))

	// MsgGrantOperatorAuthorization (DE module)
	grantMsg := &dev1.MsgGrantOperatorAuthorization{
		Authority: "verana1f06a8j6n02ash0vtkdge0a9pvvfa5ypghkyttt",
		Operator:  "",
		Grantee:   "verana13627hsukut3p39l5lawa88er0az36227rkwt83",
		MsgTypes: []string{
			"/verana.tr.v1.MsgCreateTrustRegistry",
			"/verana.tr.v1.MsgUpdateTrustRegistry",
		},
	}
	bz6, _ := enc.Marshal(grantMsg)
	fmt.Printf("MsgGrantOperatorAuthorization value:\n%s\n\n", string(bz6))

	// Now test the amino name resolution
	// The aminojson encoder resolves amino names from:
	// 1. amino.name proto option
	// 2. Proto message full name (fallback)
	fmt.Println("Proto message full names (used as amino type when no amino.name):")
	fmt.Printf("  MsgCreateTrustRegistry: %s\n", msg.ProtoReflect().Descriptor().FullName())
	fmt.Printf("  MsgUpdateTrustRegistry: %s\n", updateMsg.ProtoReflect().Descriptor().FullName())
	fmt.Printf("  MsgArchiveTrustRegistry: %s\n", archiveMsg.ProtoReflect().Descriptor().FullName())
	fmt.Printf("  MsgAddGovernanceFrameworkDocument: %s\n", addGfdMsg.ProtoReflect().Descriptor().FullName())
	fmt.Printf("  MsgIncreaseActiveGovernanceFrameworkVersion: %s\n", increaseMsg.ProtoReflect().Descriptor().FullName())
	fmt.Printf("  MsgGrantOperatorAuthorization: %s\n", grantMsg.ProtoReflect().Descriptor().FullName())
}
