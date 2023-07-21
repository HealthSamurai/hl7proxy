#!/bin/bash

# Make sure at least one argument is passed
if [ "$#" -eq 0 ]; then
    echo "Please provide at least one file as argument."
    exit 1
fi

# Create an empty string to hold the file content
content=""

# Iterate over the arguments
for file in "$@"
do
    # Check if file exists
    if [ -f "$file" ]; then
        # Append to the content string
        content+="\x0b$(cat "$file")\x1c\x0d"
    else
        echo "File $file does not exist."
        exit 1
    fi
done

# Send the content to nc command
echo -n -e "$content" | nc localhost 4242 | tr -d '\013\034' | tr '\015' '\n'
