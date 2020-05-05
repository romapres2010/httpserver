#!/bin/bash

for (( i=1; i<=$3; i++ ))
do
   ab -k -c $2 -n $4 "http://"$1"/depts/random" > ab_"$2"_parallel_"$i".txt &
done
