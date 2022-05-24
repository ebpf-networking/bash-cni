#!/bin/sh

cp /bridge /opt/cni/bin/bridge
cp /bash-cni /opt/cni/bin/bash-cni

mkdir -p /opt/cni/xdp
cp /app/xdp-loader /opt/cni/xdp
cp /app/bpftool /opt/cni/xdp
cp /app/xdp_stats /opt/cni/xdp
cp /app/xdp_kern.o /opt/cni/xdp
