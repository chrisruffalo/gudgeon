#!/bin/bash

# change ownership of directories
chown -R :gudgeon /etc/gudgeon
chown -R gudgeon:gudgeon /var/lib/gudgeon

# mod gudgeon user for files created/owned by install
usermod gudgeon -d /var/lib/gudgeon || true

# reload daemon files
systemctl daemon-reload

# restart service only if it is already running to pick up the new version
IS_RUNNING=$(systemctl is-active gudgeon)
if [[ "active" == "${IS_RUNNING}" ]]; then
    systemctl restart gudgeon
fi