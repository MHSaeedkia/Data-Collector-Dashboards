#!/bin/sh
while true; do
  echo "Fetching new NiFi token..."
  curl -sk -d "username=${NIFI_USER}&password=${NIFI_PASS}" \
    "${NIFI_URL}/nifi-api/access/token" -o /secrets/nifi_token.txt
  if [ -s /secrets/nifi_token.txt ]; then
    echo "Token updated successfully at $(date)"
  else
    echo "ERROR: token file is empty or missing at $(date)"
  fi
  sleep "${REFRESH_INTERVAL}"
done