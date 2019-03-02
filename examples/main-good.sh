#!/bin/sh
. /usr/src/app/.bashrc

echo "starting main.sh"

echo $APP_HOME
n=1

# continue until $n equals 5
while [ $n -le 100 ]
do
	echo "Welcome $n times."
	n=$(( n+1 ))	 # increments $n
    sleep 3
done

echo "complete."

exit 0
