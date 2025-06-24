#!/bin/bash

# 订阅转换服务启动脚本
# 适用于宝塔面板守护进程

# 设置工作目录
cd /www/subscription-converter/

# 设置环境变量（如果需要）
export PATH=$PATH:/usr/local/bin

# 启动程序
exec ./subscription-converter-linux-amd64 