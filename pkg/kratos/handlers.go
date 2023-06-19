package kratos

import (
	"context"
	"log"
	"net/http"
	"strconv"

	hydra_client "github.com/ory/hydra-client-go/v2"
	kratos_client "github.com/ory/kratos-client-go"
)

type API struct {
	kratos KratosClientInterface
	hydra  HydraClientInterface
}

func (a *API) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/api/kratos/self-service/login/browser", a.handleCreateFlow)
	mux.HandleFunc("/api/kratos/self-service/login/flows", a.handleLoginFlow)
	mux.HandleFunc("/api/kratos/self-service/login", a.handleUpdateFlow)
	mux.HandleFunc("/api/kratos/self-service/errors", a.handleKratosError)
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleCreateFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// We try to see if the user is logged in, because if they are the CreateBrowserLoginFlow
	// call will return an empty response
	// TODO: We need to send a different content-type to CreateBrowserLoginFlow in order
	// to avoid this bug.
	if c, _ := r.Cookie("ory_kratos_session"); c != nil {
		session, session_resp, e := a.kratos.FrontendApi().ToSession(context.Background()).
			Cookie(cookiesToString(r.Cookies())).
			Execute()
		if session_resp.StatusCode != 401 {
			if e != nil {
				log.Printf("Error when calling `FrontendApi.ToSession`: %v\n", e)
				log.Printf("Full HTTP response: %v\n", session_resp)
			} else {
				accept := hydra_client.NewAcceptOAuth2LoginRequest(session.Identity.Id)

				_, resp, e := a.hydra.OAuth2Api().AcceptOAuth2LoginRequest(context.Background()).
					LoginChallenge(q.Get("login_challenge")).
					AcceptOAuth2LoginRequest(*accept).
					Execute()
				if e != nil {
					log.Printf("Error when calling `AdminApi.AcceptLoginRequest`: %v\n", e)
					log.Printf("Full HTTP response: %v\n", resp)
					return
				}

				log.Println(resp.Body)
				writeResponse(w, resp)

				return
			}
		}
	}

	refresh, err := strconv.ParseBool(q.Get("refresh"))
	if err == nil {
		refresh = false
	}

	// We redirect the user back to this endpoint with the login_challenge, after they log in, to bypass
	// Kratos bug where the user is not redirected to hydra the first time they log in.
	// Relevant issue https://github.com/ory/kratos/issues/3052
	_, resp, e := a.kratos.FrontendApi().
		CreateBrowserLoginFlow(context.Background()).
		Aal(q.Get("aal")).
		ReturnTo(q.Get("return_to")).
		LoginChallenge(q.Get("login_challenge")).
		Refresh(refresh).
		ReturnTo(getBaseURL(r) + "/login?login_challenge=" + q.Get("login_challenge")).
		Cookie(cookiesToString(r.Cookies())).
		Execute()
	if e != nil {
		log.Printf("Error when calling `FrontendApi.CreateBrowserLoginFlow`: %v\n", e)
		log.Printf("Full HTTP response: %v\n", resp)
		return
	}

	writeResponse(w, resp)

	return
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleLoginFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	_, resp, e := a.kratos.FrontendApi().
		GetLoginFlow(context.Background()).
		Id(q.Get("id")).
		Cookie(cookiesToString(r.Cookies())).
		Execute()
	if e != nil && resp.StatusCode != 422 {
		log.Printf("Error when calling `FrontendApi.GetLoginFlow`: %v\n", e)
		log.Printf("Full HTTP response: %v\n", resp)
		return
	}

	writeResponse(w, resp)

	return
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleUpdateFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	body := new(kratos_client.UpdateLoginFlowWithOidcMethod)
	parseBody(r, body)

	_, resp, e := a.kratos.FrontendApi().
		UpdateLoginFlow(context.Background()).
		Flow(q.Get("flow")).
		UpdateLoginFlowBody(
			kratos_client.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(
				body,
			),
		).
		Cookie(cookiesToString(r.Cookies())).
		Execute()
	if e != nil && resp.StatusCode != 422 {
		log.Printf("Error when calling `FrontendApi.UpdateLoginFlow`: %v\n", e)
		log.Printf("Full HTTP response: %v\n", resp)
		return
	}

	writeResponse(w, resp)

	return
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleKratosError(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")

	_, resp, e := a.kratos.FrontendApi().GetFlowError(context.Background()).Id(id).Execute()
	if e != nil {
		log.Printf("Error when calling `FrontendApi.GetFlowError`: %v\n", e)
		log.Printf("Full HTTP response: %v\n", resp)
		return
	}
	writeResponse(w, resp)
	return
}

func NewAPI(kratos KratosClientInterface, hydra HydraClientInterface) *API {
	a := new(API)

	a.kratos = kratos
	a.hydra = hydra

	return a
}
