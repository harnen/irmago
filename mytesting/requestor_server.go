package main

import (
        "github.com/privacybydesign/irmago/server"
	"github.com/privacybydesign/irmago/server/requestorserver"
	irma "github.com/privacybydesign/irmago"

	"github.com/sirupsen/logrus"
	
	"fmt"
	"path/filepath"
	
)

var (
//	httpServer              *http.Server
//	irmaServer              *irmaserver.Server
//	irmaServerConfiguration *server.Configuration
//	requestorServer         *requestorserver.Server

	logger   = logrus.New()
	testdata = "/home/krol/go/src/github.com/privacybydesign/irmago/testdata"
)

func main() {
	//revocationTestCred  := irma.NewCredentialTypeIdentifier("irma-demo.MijnOverheid.root")
	//revKeyshareTestAttr      := irma.NewAttributeTypeIdentifier("test.test.email.email")
	//revKeyshareTestCred      := revKeyshareTestAttr.CredentialTypeIdentifier()
	//addr := "192.168.2.103"
	//addr := "0.0.0.0"
	//addr := "localhost"
	//addr1 := "0.0.0.0"
    var configuration = &requestorserver.Configuration{
        Configuration: &server.Configuration{
                URL:                   "http://localhost:48682/irma",
                Logger:                logger,
                DisableSchemesUpdate:  true,
                SchemesPath:           filepath.Join(testdata, "irma_configuration"),
                IssuerPrivateKeysPath: filepath.Join(testdata, "privatekeys"),
                JwtPrivateKeyFile: filepath.Join(testdata, "jwtkeys", "sk.pem"),
                StaticSessions: map[string]interface{}{
                        "staticsession": irma.ServiceProviderRequest{
                                RequestorBaseRequest: irma.RequestorBaseRequest{
                                        CallbackURL: "http://localhost:48685",
                                },
                                Request: &irma.DisclosureRequest{
                                        BaseRequest: irma.BaseRequest{LDContext: irma.LDContextDisclosureRequest},
                                        Disclose: irma.AttributeConDisCon{
                                                {{irma.NewAttributeRequest("irma-demo.RU.studentCard.level")}},
                                        },
                                },
                        },
                },
        },
        //ListenAddress: "localhost",
        Port:          48682,
        DisableRequestorAuthentication: false,
        MaxRequestAge:                  3,
        Permissions: requestorserver.Permissions{
                Disclosing: []string{"*"},
                Signing:    []string{"*"},
                Issuing:    []string{"*"},
        },
        Requestors: map[string]requestorserver.Requestor{
                "requestor1": {
                        AuthenticationMethod:  requestorserver.AuthenticationMethodPublicKey,
                        AuthenticationKeyFile: filepath.Join(testdata, "jwtkeys", "requestor1.pem"),
                },
                "requestor2": {
                        AuthenticationMethod: requestorserver.AuthenticationMethodToken,
                        AuthenticationKey:    "xa6=*&9?8jeUu5>.f-%rVg`f63pHim",
                },
                "requestor3": {
                        AuthenticationMethod: requestorserver.AuthenticationMethodHmac,
                        AuthenticationKey:    "eGE2PSomOT84amVVdTU+LmYtJXJWZ2BmNjNwSGltCg==",
                },
        },
	}

	fmt.Println("Created config")
	var err error
	var requestorServer *requestorserver.Server
	if requestorServer, err = requestorserver.New(configuration); err != nil {
		panic("Creating server failed:" + err.Error())
	}
	if err = requestorServer.Start(configuration); err != nil {
		panic("Starting server failed: " + err.Error())
	}
}
