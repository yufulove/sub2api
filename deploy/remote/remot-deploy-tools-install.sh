#!/bin/bash

### remot_deploy_tools_install.sh

cd /home
mkdir -p /home/work/tools
cd /home/work/tools

sudo yum install -y git
sudo dnf install -y  nodejs npm

wget https://go.dev/dl/go1.26.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.2.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc


mkdir -p /home/work/deploy
cd /home/work/deploy
git clone -b main https://github.com/yufenfei2026/deploy-sub2api.git
git clone -b main https://github.com/yufulove/sub2api.git

