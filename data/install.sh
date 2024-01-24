#!/bin/bash

if [ "$1" == "uninstall" ]; then
    rm -f /usr/local/bin/openess
    rm -f /etc/systemd/system/openess.service
    rm -Rf /etc/openess
else
    install -m 755 ../openess /usr/local/bin
    install openess.service /etc/systemd/system/
    install -d /etc/openess/
    install *.json /etc/openess/
    sed -i 's/data\//\/etc\/openess\//g' /etc/openess/config.json
fi
