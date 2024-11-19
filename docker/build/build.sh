#!/bin/bash
cd /app && \
git clone https://github.com/feishu/kun.git /app/kun && \
git clone https://github.com/feishu/xun.git /app/xun && \
git clone https://github.com/feishu/gou.git /app/gou && \
git clone https://github.com/feishu/v8go2.git /app/v8go && \
git clone https://github.com/feishu/xgen.git /app/xgen-v1.0 && \
git clone https://github.com/feishu/yao-init.git /app/yao-init && \
git clone https://github.com/feishu/yao.git /app/yao


cd /app/yao && \
export VERSION=$(cat share/const.go  |grep 'const VERSION' | awk '{print $4}' | sed "s/\"//g") 

cd /app/yao && make tools && make artifacts-linux
mv /app/yao/dist/release/* /data/
