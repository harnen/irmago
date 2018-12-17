package irma

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"bytes"

	"fmt"

	"github.com/go-errors/errors"
	"github.com/mhe/gabi"
)

// Status encodes the status of an IRMA session (e.g., connected).
type Status string

var ForceHttps bool = true

const (
	MinVersionHeader = "X-IRMA-MinProtocolVersion"
	MaxVersionHeader = "X-IRMA-MaxProtocolVersion"
)

// ProtocolVersion encodes the IRMA protocol version of an IRMA session.
type ProtocolVersion struct {
	Major int
	Minor int
}

func NewVersion(major, minor int) *ProtocolVersion {
	return &ProtocolVersion{major, minor}
}

func (v *ProtocolVersion) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

func (v *ProtocolVersion) UnmarshalJSON(b []byte) (err error) {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		str = string(b) // If b is not enclosed by quotes, try it directly
	}
	parts := strings.Split(str, ".")
	if len(parts) != 2 {
		return errors.New("Invalid protocol version number: not of form x.y")
	}
	if v.Major, err = strconv.Atoi(parts[0]); err != nil {
		return
	}
	v.Minor, err = strconv.Atoi(parts[1])
	return
}

func (v *ProtocolVersion) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

// Returns true if v is below the given version.
func (v *ProtocolVersion) Below(major, minor int) bool {
	if v.Major < major {
		return true
	}
	return v.Major == major && v.Minor < minor
}

func (v *ProtocolVersion) BelowVersion(other *ProtocolVersion) bool {
	return v.Below(other.Major, other.Minor)
}

func (v *ProtocolVersion) Above(major, minor int) bool {
	if v.Major > major {
		return true
	}
	return v.Major == major && v.Minor > minor
}

func (v *ProtocolVersion) AboveVersion(other *ProtocolVersion) bool {
	return v.Above(other.Major, other.Minor)
}

// GetMetadataVersion maps a chosen protocol version to a metadata version that
// the server will use.
func GetMetadataVersion(v *ProtocolVersion) byte {
	if v.Below(2, 3) {
		return 0x02 // no support for optional attributes
	}
	return 0x03 // current version
}

// Action encodes the session type of an IRMA session (e.g., disclosing).
type Action string

// ErrorType are session errors.
type ErrorType string

// SessionError is a protocol error.
type SessionError struct {
	Err error
	ErrorType
	Info         string
	RemoteError  *RemoteError
	RemoteStatus int
}

// RemoteError is an error message returned by the API server on errors.
type RemoteError struct {
	Status      int    `json:"status"`
	ErrorName   string `json:"error"`
	Description string `json:"description"`
	Message     string `json:"message"`
	Stacktrace  string `json:"stacktrace"`
}

type Validator interface {
	Validate() error
}

// UnmarshalValidate json.Unmarshal's data, and validates it using the
// Validate() method if dest implements the Validator interface.
func UnmarshalValidate(data []byte, dest interface{}) error {
	if err := json.Unmarshal(data, dest); err != nil {
		return err
	}
	if v, ok := dest.(Validator); ok {
		return v.Validate()
	}
	return nil
}

func (err *RemoteError) Error() string {
	var msg string
	if err.Message != "" {
		msg = fmt.Sprintf(" (%s)", err.Message)
	}
	return fmt.Sprintf("%s%s: %s", err.ErrorName, msg, err.Description)
}

// Qr contains the data of an IRMA session QR (as generated by irma_js),
// suitable for NewSession().
type Qr struct {
	// Server with which to perform the session
	URL string `json:"u"`
	// Session type (disclosing, signing, issuing)
	Type Action `json:"irmaqr"`
}

type SchemeManagerRequest Qr

// Statuses
const (
	StatusConnected     = Status("connected")
	StatusCommunicating = Status("communicating")
	StatusManualStarted = Status("manualStarted")
)

// Actions
const (
	ActionSchemeManager = Action("schememanager")
	ActionDisclosing    = Action("disclosing")
	ActionSigning       = Action("signing")
	ActionIssuing       = Action("issuing")
	ActionUnknown       = Action("unknown")
)

// Protocol errors
const (
	// Protocol version not supported
	ErrorProtocolVersionNotSupported = ErrorType("protocolVersionNotSupported")
	// Error in HTTP communication
	ErrorTransport = ErrorType("transport")
	// Invalid client JWT in first IRMA message
	ErrorInvalidJWT = ErrorType("invalidJwt")
	// Unkown session type (not disclosing, signing, or issuing)
	ErrorUnknownAction = ErrorType("unknownAction")
	// Crypto error during calculation of our response (second IRMA message)
	ErrorCrypto = ErrorType("crypto")
	// Server rejected our response (second IRMA message)
	ErrorRejected = ErrorType("rejected")
	// (De)serializing of a message failed
	ErrorSerialization = ErrorType("serialization")
	// Error in keyshare protocol
	ErrorKeyshare = ErrorType("keyshare")
	// API server error
	ErrorApi = ErrorType("api")
	// Server returned unexpected or malformed response
	ErrorServerResponse = ErrorType("serverResponse")
	// Credential type not present in our Configuration
	ErrorUnknownCredentialType = ErrorType("unknownCredentialType")
	// Error during downloading of credential type, issuer, or public keys
	ErrorConfigurationDownload = ErrorType("configurationDownload")
	// IRMA requests refers to unknown scheme manager
	ErrorUnknownSchemeManager = ErrorType("unknownSchemeManager")
	// A session is requested involving a scheme manager that has some problem
	ErrorInvalidSchemeManager = ErrorType("invalidSchemeManager")
	// Recovered panic
	ErrorPanic = ErrorType("panic")
)

func (e *SessionError) Error() string {
	var buffer bytes.Buffer
	typ := e.ErrorType
	if typ == "" {
		typ = ErrorType("unknown")
	}

	buffer.WriteString("Error type: ")
	buffer.WriteString(string(typ))
	if e.Err != nil {
		buffer.WriteString("\nDescription: ")
		buffer.WriteString(e.Err.Error())
	}
	if e.RemoteStatus != 200 {
		buffer.WriteString("\nStatus code: ")
		buffer.WriteString(strconv.Itoa(e.RemoteStatus))
	}
	if e.RemoteError != nil {
		buffer.WriteString("\nIRMA server error: ")
		buffer.WriteString(e.RemoteError.Error())
	}

	return buffer.String()
}

func (e *SessionError) WrappedError() string {
	if e.Err == nil {
		return ""
	}

	return e.Err.Error()
}

func (e *SessionError) Stack() string {
	if withStack, ok := e.Err.(*errors.Error); ok {
		return string(withStack.Stack())
	}

	return ""
}

type Disclosure struct {
	Proofs  gabi.ProofList            `json:"proofs"`
	Indices DisclosedAttributeIndices `json:"indices"`
}

// DisclosedAttributeIndices contains, for each conjunction of an attribute disclosure request,
// a list of attribute indices, pointing to where the disclosed attributes for that conjunction
// can be found within a gabi.ProofList.
type DisclosedAttributeIndices [][]*DisclosedAttributeIndex

// DisclosedAttributeIndex points to a specific attribute in a gabi.ProofList.
type DisclosedAttributeIndex struct {
	CredentialIndex int                  `json:"cred"`
	AttributeIndex  int                  `json:"attr"`
	Identifier      CredentialIdentifier `json:"-"` // credential from which this attribute was disclosed
}

type IssueCommitmentMessage struct {
	*gabi.IssueCommitmentMessage
	Indices DisclosedAttributeIndices `json:"indices"`
}

func (i *IssueCommitmentMessage) Disclosure() *Disclosure {
	return &Disclosure{
		Proofs:  i.Proofs,
		Indices: i.Indices,
	}
}

func JwtDecode(jwt string, body interface{}) error {
	jwtparts := strings.Split(jwt, ".")
	if jwtparts == nil || len(jwtparts) < 2 {
		return errors.New("Not a JWT")
	}
	bodybytes, err := base64.RawStdEncoding.DecodeString(jwtparts[1])
	if err != nil {
		return err
	}
	return json.Unmarshal(bodybytes, body)
}

func ParseRequestorJwt(action string, jwt string) (RequestorJwt, error) {
	var retval RequestorJwt
	switch action {
	case "verification_request", string(ActionDisclosing):
		retval = &ServiceProviderJwt{}
	case "signature_request", string(ActionSigning):
		retval = &SignatureRequestorJwt{}
	case "issue_request", string(ActionIssuing):
		retval = &IdentityProviderJwt{}
	default:
		return nil, errors.New("Invalid session type")
	}
	err := JwtDecode(jwt, retval)
	if err != nil {
		return nil, err
	}
	return retval, nil
}

func (qr *Qr) Validate() (err error) {
	if qr.URL == "" {
		return errors.New("No URL specified")
	}
	var u *url.URL
	if u, err = url.ParseRequestURI(qr.URL); err != nil {
		return errors.Errorf("Invalid URL: %s", err.Error())
	}
	if ForceHttps && u.Scheme != "https" {
		return errors.Errorf("URL did not begin with https")
	}

	switch qr.Type {
	case ActionDisclosing: // nop
	case ActionIssuing: // nop
	case ActionSigning: // nop
	default:
		return errors.New("Unsupported session type")
	}

	return nil
}

func (smr *SchemeManagerRequest) Validate() error {
	if smr.Type != ActionSchemeManager {
		return errors.New("Not a scheme manager request")
	}
	if smr.URL == "" {
		return errors.New("No URL specified")
	}
	if _, err := url.ParseRequestURI(smr.URL); err != nil {
		return errors.Errorf("Invalid URL: %s", err.Error())
	}
	return nil
}
