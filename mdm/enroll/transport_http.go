package enroll

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/micromdm/micromdm/pkg/crypto"

	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/groob/plist"
	"go.mozilla.org/pkcs7"
)

type HTTPHandlers struct {
	EnrollHandler    http.Handler
	OTAEnrollHandler http.Handler

	// In Apple's Over-the-Air design Phases 2 and 3 happen over the same URL.
	// The differentiator is which certificate signed the CMS POST body.
	OTAPhase2Phase3Handler http.Handler
}

type OSTooOldError struct {
	Code        string `json:"code" plist:"code"`
	Description string `json:"description,omitempty" plist:"description,omitempty"`
	Message     string `json:"message,omitempty" plist:"message,omitempty"`
	Details     struct {
		OSVersion    string `json:"OSVersion" plist:"OSVersion"`
		BuildVersion string `json:"BuildVersion,omitempty" plist:"BuildVersion"`
	} `json:"details" plist:"details"`
}

// error conformance
func (e *OSTooOldError) Error() string {
	return "this OS is too old"
}

func MakeHTTPHandlers(ctx context.Context, endpoints Endpoints, v *crypto.PKCS7Verifier, opts ...httptransport.ServerOption) HTTPHandlers {
	ver := verifier{PKCS7Verifier: v}
	enrollmentOpts := append(opts, httptransport.ServerErrorEncoder(encodeUpdateRequiredError))

	h := HTTPHandlers{
		EnrollHandler: httptransport.NewServer(
			endpoints.GetEnrollEndpoint,
			ver.decodeMDMEnrollRequest,
			encodeMobileconfigResponse,
			enrollmentOpts...,
		),
		OTAEnrollHandler: httptransport.NewServer(
			endpoints.OTAEnrollEndpoint,
			decodeEmptyRequest,
			encodeMobileconfigResponse,
			opts...,
		),
		OTAPhase2Phase3Handler: httptransport.NewServer(
			endpoints.OTAPhase2Phase3Endpoint,
			ver.decodeOTAPhase2Phase3Request,
			encodeMobileconfigResponse,
			opts...,
		),
	}
	return h
}

func decodeEmptyRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

type verifier struct {
	*crypto.PKCS7Verifier
}

func (v verifier) decodeMDMEnrollRequest(_ context.Context, r *http.Request) (interface{}, error) {
	switch r.Method {
	case "GET":
		return mdmEnrollRequest{}, nil
	case "POST": // DEP request
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		p7, err := pkcs7.Parse(data)
		if err != nil {
			return nil, err
		}
		err = v.Verify(p7)
		if err != nil {
			return nil, err
		}
		// TODO: for thse errors provide better feedback as 4xx HTTP status
		signer := p7.GetOnlySigner()
		if signer == nil {
			return nil, errors.New("invalid CMS signer during enrollment")
		}
		err = crypto.VerifyFromAppleDeviceCA(signer)
		if err != nil {
			return nil, errors.New("unauthorized enrollment client: not signed by Apple Device CA")
		}
		var request depEnrollmentRequest
		if err := plist.Unmarshal(p7.Content, &request); err != nil {
			return nil, err
		}
		return request, nil
	default:
		return nil, errors.New("unknown enrollment method")
	}
}

func encodeUpdateRequiredError(ctx context.Context, err error, w http.ResponseWriter) {
	switch err.(type) {
	case *OSTooOldError:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		resp, _ := json.Marshal(err)
		w.Write(resp)

	default:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func encodeMobileconfigResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	mcResp := response.(mobileconfigResponse)
	_, err := w.Write(mcResp.Mobileconfig)
	return err
}

func (v verifier) decodeOTAPhase2Phase3Request(_ context.Context, r *http.Request) (interface{}, error) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	p7, err := pkcs7.Parse(data)
	if err != nil {
		return nil, err
	}
	err = v.Verify(p7)
	if err != nil {
		return nil, err
	}
	var request otaEnrollmentRequest
	err = plist.Unmarshal(p7.Content, &request)
	if err != nil {
		return nil, err
	}
	return mdmOTAPhase2Phase3Request{request, p7}, nil
}
