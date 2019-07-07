#!/bin/sh
set -e

while true; do
    if [ -f /var/run/haproxy.pid ]; then
        if [ -f /app/src/haproxy.tmp.cfg ]; then
            echo '[START] Reload haproxy : pid '$(cat /var/run/haproxy.pid) ;
            cp /app/src/haproxy.tmp.cfg /app/src/haproxy.cfg
            rm /app/src/haproxy.tmp.cfg
            kill -HUP $(cat /var/run/haproxy.pid) ;
            echo '[SUCCESS] Reload haproxy' ;
        fi
    else
        if [ -f /app/src/haproxy.tmp.cfg ]; then
            echo '[START] Start Haproxy' ;
            cp /app/src/haproxy.tmp.cfg /app/src/haproxy.cfg
            rm /app/src/haproxy.tmp.cfg
            haproxy -W -D -f /app/src/haproxy.cfg -p /var/run/haproxy.pid &
            echo '[SUCCESS] Start Haproxy' ;
        fi
    fi
    sleep 1
done
