#!/bin/bash
for P in $(go list ./... | grep -v vendor/); do 
	echo go vet $P
	go vet $P
	if [ $? -eq 1 ]; then
		echo ""
		echo "Vet found suspicious constructs. Please check the reported constructs"
		echo "and fix them if necessary before submitting the code for review."
		exit 1
	fi
done
