#!/bin/bash

echo "Generating keypair"
irma scheme issuer keygen -f -l 2048 -c 3
irma scheme issuer keygen -f -l 2048 -c 4

echo "Removing keys from ./testdata/privatekeys/"
rm ./testdata/privatekeys/*


echo "Replacing issuers pub/priv keys"
cp ./PublicKeys/3.xml ./testdata/irma_configuration/irma-demo/MijnOverheid/PublicKeys/
cp ./PublicKeys/3.xml ./testdata/irma_configuration/irma-demo/RU/PublicKeys/
cp ./PublicKeys/4.xml ./testdata/irma_configuration/test/test/PublicKeys/

cp ./PrivateKeys/3.xml ./testdata/irma_configuration/irma-demo/MijnOverheid/PrivateKeys/
cp ./PrivateKeys/3.xml ./testdata/irma_configuration/irma-demo/RU/PrivateKeys/
cp ./PrivateKeys/4.xml ./testdata/irma_configuration/test/test/PrivateKeys/

echo "Signing schemes"
cd ./testdata/irma_configuration/irma-demo/
irma scheme sign

cd ../test/
irma scheme sign

