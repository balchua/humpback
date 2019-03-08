#!/bin/sh
n=1

# continue until $n equals 5
while [ $n -le 10 ]
do
	echo "Welcome $n times."
	./humpback --application app1-1gb-mem -k $KUBECONFIG -n default -c /usr/src/app/main-good.sh &
        n=$(( n+1 ))	 # increments $n
done

echo "complete."

exit 0
