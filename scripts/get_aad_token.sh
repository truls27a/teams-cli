#!/usr/bin/env bash
set -euo pipefail

CLIENT_ID="1fec8e78-bce4-4aaf-ab1b-5451cc387264"
TENANT="${TEAMS_AAD_TENANT:-organizations}"
SCOPE="https://api.spaces.skype.com/.default offline_access"

dc=$(curl -fsS -X POST \
  "https://login.microsoftonline.com/${TENANT}/oauth2/v2.0/devicecode" \
  --data-urlencode "client_id=${CLIENT_ID}" \
  --data-urlencode "scope=${SCOPE}")

device_code=$(jq -r .device_code <<<"$dc")
user_code=$(jq -r .user_code <<<"$dc")
verification_uri=$(jq -r .verification_uri <<<"$dc")
interval=$(jq -r .interval <<<"$dc")
expires_in=$(jq -r .expires_in <<<"$dc")

>&2 echo
>&2 echo "  Open: $verification_uri"
>&2 echo "  Code: $user_code"
>&2 echo

deadline=$(( $(date +%s) + expires_in ))
while [ "$(date +%s)" -lt "$deadline" ]; do
  sleep "$interval"
  resp=$(curl -sS -X POST \
    "https://login.microsoftonline.com/${TENANT}/oauth2/v2.0/token" \
    --data-urlencode "client_id=${CLIENT_ID}" \
    --data-urlencode "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
    --data-urlencode "device_code=${device_code}")
  err=$(jq -r '.error // empty' <<<"$resp")
  case "$err" in
    "") break ;;
    authorization_pending) ;;
    slow_down) interval=$((interval + 5)) ;;
    *) >&2 echo "error: $err"; exit 1 ;;
  esac
done

if [ -z "${resp:-}" ] || [ -n "$(jq -r '.error // empty' <<<"$resp")" ]; then
  >&2 echo "device code expired"
  exit 1
fi

now=$(date +%s)
expires_at=$(( now + $(jq -r .expires_in <<<"$resp") ))

echo "TEAMS_AAD_TENANT=${TENANT}"
echo "TEAMS_AAD_EXPIRES_AT=${expires_at}"
echo "TEAMS_AAD_ACCESS_TOKEN=$(jq -r .access_token <<<"$resp")"
refresh=$(jq -r '.refresh_token // empty' <<<"$resp")
[ -n "$refresh" ] && echo "TEAMS_AAD_REFRESH_TOKEN=${refresh}"
