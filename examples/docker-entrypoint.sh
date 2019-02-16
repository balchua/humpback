#!/bin/sh
. /usr/src/app/.bashrc
export APP_HOME=/root

echo "command executed : $@"

echo $APP_HOME

exec $@