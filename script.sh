#!/bin/bash

echo "Running 100 clients..."
for i in {1..100}; do
    make client CLIENT_ID=$i &
done
