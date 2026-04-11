#!/bin/bash

### remote-01-tools-install.sh

cd /home
mkdir -p /home/work/tools
cd /home/work/tools

#sudo yum install -y git
if ! command -v git &> /dev/null; then
    echo "Git 未安装，正在安装..."
    yum install -y git
    if [ $? -eq 0 ]; then
        echo "Git 安装成功"
        git --version
    else
        echo "Git 安装失败"
        exit 1
    fi
fi

sudo dnf install -y  nodejs npm

wget https://go.dev/dl/go1.26.2.linux-amd64.tar.gz

sudo tar -C /usr/local -xzf go1.26.2.linux-amd64.tar.gz

echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

mkdir -p /home/work/deploy
cd /home/work/deploy

#sub2api项目路径及下载项目代码
git clone -b prod https://github.com/yufulove/sub2api.git

# 制品包git项目路径及下载项目代码
git clone -b prod https://github.com/yufenfei2026/deploy-sub2api.git



