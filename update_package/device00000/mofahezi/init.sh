#!/bin/bash
# 删除root目录下的没用的文件
cd /root
rm -rf *
chmod 666 /mofahezi
chmod +x /mofahezi/check.out
cd /mofahezi
nohup ./check.out >/dev/null 2>&1 &