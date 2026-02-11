#!/bin/bash
# Create SealedSecret for Discord bot credentials.
# Run this after creating a Discord bot at https://discord.com/developers
#
# Prerequisites:
# 1. Create a Discord Application + Bot
# 2. Enable MESSAGE CONTENT Intent in Bot settings
# 3. Invite bot to server with Send Messages + Read Message History permissions
# 4. Copy Bot Token and Channel ID

set -euo pipefail

read -rp "Discord Bot Token: " DISCORD_BOT_TOKEN
read -rp "Discord Channel ID: " DISCORD_CHANNEL_ID

kubectl create secret generic factorio-discord-secrets \
  --namespace=factorio \
  --from-literal=discord-bot-token="$DISCORD_BOT_TOKEN" \
  --from-literal=discord-channel-id="$DISCORD_CHANNEL_ID" \
  --dry-run=client -o yaml \
  | kubeseal --format yaml > sealed-secret.yaml

echo "Created sealed-secret.yaml"
echo "Commit and push to deploy."
