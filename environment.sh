#!/bin/bash
groupadd -g 801 parsley
useradd -M -u 1201 -g parsley parsley
chown -R parsley:parsley /parsley
echo 'export PYTHONIOENCODING=utf-8' >> ~/.bashrc