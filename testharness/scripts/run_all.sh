#!/bin/bash
# scripts/run_all.sh

# Exit immediately if a command exits with a non-zero status
set -e

# Ecosystem Operator Authorization Journeys
echo "*****************************************************************************************"
echo "************************ Running test-harness journey 101 *******************************"
echo "*****************************************************************************************"
go run cmd/main.go 101

echo "*****************************************************************************************"
echo "************************ Running test-harness journey 102 *******************************"
echo "*****************************************************************************************"
go run cmd/main.go 102

# Credential Schema Operator Authorization Journeys
echo "*****************************************************************************************"
echo "************************ Running test-harness journey 201 *******************************"
echo "*****************************************************************************************"
go run cmd/main.go 201

echo "*****************************************************************************************"
echo "************************ Running test-harness journey 202 *******************************"
echo "*****************************************************************************************"
go run cmd/main.go 202

# Permission Operator Authorization Journeys
echo "*****************************************************************************************"
echo "************************ Running test-harness journey 301 *******************************"
echo "*****************************************************************************************"
go run cmd/main.go 301

# Journeys 302-310 rely on the legacy "operator self-delegation shortcut"
# where a regular wallet acts as the `corporation` field. Under spec v4-rc2
# (issue #305) AUTHZ-CHECK-5 requires `corporation` to be the policy_address
# of a registered MOD-CO Corporation, so the shortcut no longer works — those
# journeys need to be rewritten to use the j301 group + group-proposal flow.
# Tracked as follow-up; not part of the TR→EC rename PR.
echo "*****************************************************************************************"
echo "******** Journeys 302-310 SKIPPED — pending rewrite for AUTHZ-CHECK-5 (see PR) **********"
echo "*****************************************************************************************"
