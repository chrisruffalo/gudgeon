#!/bin/bash

# add gudgeon user
useradd gudgeon -b /var/lib/gudgeon -s /sbin/nologin || true

# change ownership of directories
chown -R :gudgeon /etc/gudgeon
chown -R gudgeon:gudgeon /var/lib/gudgeon