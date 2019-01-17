#!/bin/bash

# add user before install if user does not exist
USER_EXISTS=$(id -u gudgeon)
if ["0" == "$?"]; then
    useradd gudgeon -s /sbin/nologin || true
fi