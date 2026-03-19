#!/bin/bash
# OpenClaw Helper - Ubuntu Configuration (fallback)
set -e

##OCH:PROGRESS:10:Updating package lists...
sudo apt-get update -y

##OCH:PROGRESS:50:Installing build dependencies...
sudo apt-get install -y curl git build-essential

##OCH:PROGRESS:100:Ubuntu configured
