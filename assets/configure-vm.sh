#!/bin/bash

# install dependencies
sudo apt-get update
sudo apt-get install -y git python3 python3-pip

# get log4j-shell-poc ready
ssh-keygen -F github.com || ssh-keyscan github.com >>~/.ssh/known_hosts
git clone git@github.com:kozmer/log4j-shell-poc.git
cd log4j-shell-poc
pip3 install -r requirements.txt
