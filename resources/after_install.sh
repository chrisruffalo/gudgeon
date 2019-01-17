#!/bin/bash

# change ownership of directories
chown -R :gudgeon /etc/gudgeon
chown -R gudgeon:gudgeon /var/lib/gudgeon

# mod gudgeon user for files created/owned by install
usermod gudgeon -d /var/lib/gudgeon || true