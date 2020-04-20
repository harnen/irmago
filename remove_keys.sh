#!/bin/bash


echo "Removing keys from ./testdata/privatekeys/"
rm ./testdata/privatekeys/*


echo "Removing issuers pub/priv keys"
rm ./testdata/irma_configuration/irma-demo/MijnOverheid/PublicKeys/3.xml
rm ./testdata/irma_configuration/irma-demo/RU/PublicKeys/3.xml
rm ./testdata/irma_configuration/test/test/PublicKeys/4.xml

rm ./testdata/irma_configuration/irma-demo/MijnOverheid/PrivateKeys/3.xml
rm ./testdata/irma_configuration/irma-demo/RU/PrivateKeys/3.xml
rm ./testdata/irma_configuration/test/test/PrivateKeys/4.xml

echo "Signing schemes"
cd ./testdata/irma_configuration/irma-demo/
irma scheme sign

cd ../test/
irma scheme sign

