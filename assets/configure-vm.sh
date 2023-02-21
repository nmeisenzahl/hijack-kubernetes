#!/bin/bash

# install dependencies
sudo apt-get update
sudo apt-get install -y git python3 python3-pip

# get log4j-shell-poc ready
git clone https://github.com/kozmer/log4j-shell-poc.git
cd log4j-shell-poc
pip3 install -r requirements.txt
