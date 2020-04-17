#!/bin/bash

echo "Generating keypair"
irma scheme issuer keygen -f -l 2048

echo "Moving private keys to ./testdata/privatekeys/"
cp ./PrivateKeys/0.xml ./testdata/privatekeys/irma-demo.MijnOverheid.xml
cp ./PrivateKeys/0.xml ./testdata/privatekeys/irma-demo.RU.xml
cp ./PrivateKeys/0.xml ./testdata/privatekeys/test.test.xml


echo "Replacing issuers pub/priv keys"
for issuer in "irma-demo/MijnOverheid" "irma-demo/RU" "test/test"
do
	rm -f ./testdata/irma_configuration/${issuer}/PublicKeys/*
	rm -f ./testdata/irma_configuration/${issuer}/PrivateKeys/*
	cp ./PublicKeys/0.xml ./testdata/irma_configuration/${issuer}/PublicKeys/
	cp ./PrivateKeys/0.xml ./testdata/irma_configuration/${issuer}/PrivateKeys/
done

echo "Signing schemes"
cd ./testdata/irma_configuration/irma-demo/
irma scheme sign

cd ../test/
irma scheme sign

