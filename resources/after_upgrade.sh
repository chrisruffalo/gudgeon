#!/bin/bash

# restart service only if it is already running to pick up the new version
IS_RUNNING=$(systemctl is-active gudgeon)
if [[ "active" == "${IS_RUNNING}" ]]; then
    systemctl restart gudgeon
fi