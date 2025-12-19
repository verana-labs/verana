#!/bin/bash

# Variables
BINARY="veranad"
KEYRING_BACKEND="test"

# Define seed phrases for different roles
SEED_PHRASE_COOLUSER="pink glory help gown abstract eight nice crazy forward ketchup skill cheese"
SEED_PHRASE_TRUST_REGISTRY_CONTROLLER="simple stuff order coach cliff advance ugly dial right forward boring rhythm comfort initial girl either universe genre pony sort own cycle hurt grit"
SEED_PHRASE_ISSUER_GRANTOR="peace load monkey fuel safe rally ship panic vapor script confirm acid size silent grit muscle olive scissors seat drift vital universe affair hero"
SEED_PHRASE_ISSUER="aim fold come benefit stuff file host joy doll grid credit garbage helmet frown rubber depart project dinosaur leisure relax equip sting flat grief"
SEED_PHRASE_VERIFIER="intact link bench vapor sense during carbon symptom grab drop ramp city life bomb ice lock mimic wine furnace often buzz muscle bird layer"
SEED_PHRASE_CREDENTIAL_HOLDER="noodle stamp flip knife pretty sail giraffe drama art addict unable curious daughter will motion team chunk seek stuff target rhythm post release piece"

# Add accounts using seed phrases
echo "Adding accounts to keyring..."

# Faucet account (cooluser)
#echo "$SEED_PHRASE_COOLUSER" | $BINARY keys add cooluser --recover --keyring-backend $KEYRING_BACKEND

# Trust Registry Controller
echo "$SEED_PHRASE_TRUST_REGISTRY_CONTROLLER" | $BINARY keys add Trust_Registry_Controller --recover --keyring-backend $KEYRING_BACKEND

# Issuer Grantor Applicant
echo "$SEED_PHRASE_ISSUER_GRANTOR" | $BINARY keys add Issuer_Grantor_Applicant --recover --keyring-backend $KEYRING_BACKEND

# Issuer Applicant
echo "$SEED_PHRASE_ISSUER" | $BINARY keys add Issuer_Applicant --recover --keyring-backend $KEYRING_BACKEND

# Verifier Applicant
echo "$SEED_PHRASE_VERIFIER" | $BINARY keys add Verifier_Applicant --recover --keyring-backend $KEYRING_BACKEND

# Credential Holder
echo "$SEED_PHRASE_CREDENTIAL_HOLDER" | $BINARY keys add Credential_Holder --recover --keyring-backend $KEYRING_BACKEND

echo "All accounts have been added to the keyring!"
echo "You can now run the test harness journeys."

# List all accounts for verification
echo "Listed accounts:"
$BINARY keys list --keyring-backend $KEYRING_BACKEND