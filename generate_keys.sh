#!/bin/bash 

rm PrivateKeys/*
rm PublicKeys/*
echo "First key"
irma scheme issuer keygen -f -l $1 -c 3
echo "Second key"
irma scheme issuer keygen -f -l $1 -c 4
