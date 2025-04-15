#!/bin/bash

for i in {1..5}; do
    echo "Starting client $i..."
    make client > logs/client_$i.log 2>&1 &
done

echo "All clients started in background."
