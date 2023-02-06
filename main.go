package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	hydra_client "github.com/ory/hydra-client-go"
	kratos_client "github.com/ory/kratos-client-go"
)

const defaultPort = "8080"

//go:embed ui/dist
//go:embed ui/dist/_next
//go:embed ui/dist/_next/static/chunks/pages/*.js
//go:embed ui/dist/_next/static/*/*.js
//go:embed ui/dist/_next/static/*/*.css
var ui embed.FS

func NewKratosClient() *kratos_client.APIClient {
	configuration := kratos_client.NewConfiguration()
	configuration.Debug = true
	kratos_url := os.Getenv("KRATOS_PUBLIC_URL")
	configuration.Servers = []kratos_client.ServerConfiguration{
		{
			URL: kratos_url,
		},
	}
	apiClient := kratos_client.NewAPIClient(configuration)
	return apiClient
}

func NewHydraClient() *hydra_client.APIClient {
	configuration := hydra_client.NewConfiguration()
	configuration.Debug = true
	hydra_url := os.Getenv("HYDRA_ADMIN_URL")
	configuration.Servers = []hydra_client.ServerConfiguration{
		{
			URL: hydra_url,
		},
	}
	apiClient := hydra_client.NewAPIClient(configuration)
	return apiClient
}

func main() {
	fs := http.FileServer(http.Dir("ui/dist"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if ext := path.Ext(r.URL.Path); ext == "" {
			r.URL.Path += ".html"
		}
		fs.ServeHTTP(w, r)
	})
	http.HandleFunc("/api/kratos/self-service/login/browser", handleCreateFlow)
	http.HandleFunc("/api/kratos/self-service/login", handleUpdateFlow)
	http.HandleFunc("/api/kratos/self-service/errors", handleKratosError)
	http.HandleFunc("/api/consent", handleConsent)

	port := os.Getenv("PORT")

	if port == "" {
		port = defaultPort
	}

	log.Println("Starting server on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleCreateFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kratos := NewKratosClient()

	// We try to see if the user is logged in, because if they are the CreateBrowserLoginFlow
	// call will return an empty response
	// TODO: We need to send a different content-type to CreateBrowserLoginFlow in order
	// to avoid this bug.
	if c, _ := r.Cookie("ory_kratos_session"); c != nil {
		session, session_resp, e := kratos.FrontendApi.ToSession(context.Background()).
			Cookie(cookiesToString(r.Cookies())).
			Execute()
		if e != nil {
			log.Printf("Error when calling `FrontendApi.ToSession`: %v\n", e)
			log.Printf("Full HTTP response: %v\n", session_resp)
			return
		}

		accept := hydra_client.NewAcceptLoginRequest(session.Identity.Id)
		hydra := NewHydraClient()
		_, resp, e := hydra.AdminApi.AcceptLoginRequest(context.Background()).
			LoginChallenge(q.Get("login_challenge")).
			AcceptLoginRequest(*accept).
			Execute()
		if e != nil {
			log.Printf("Error when calling `AdminApi.AcceptLoginRequest`: %v\n", e)
			log.Printf("Full HTTP response: %v\n", resp)
			return
		}

		log.Println(resp.Body)
		writeResponse(w, resp)
		// w.WriteHeader(200)
		// w.Write(resp.Body)

		return
	}

	refresh, err := strconv.ParseBool(q.Get("refresh"))

	if err == nil {
		refresh = false
	}
	_, resp, e := kratos.FrontendApi.
		CreateBrowserLoginFlow(context.Background()).
		Aal(q.Get("aal")).
		ReturnTo(q.Get("return_to")).
		LoginChallenge(q.Get("login_challenge")).
		Refresh(refresh).
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

func handleUpdateFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kratos := NewKratosClient()
	body := new(kratos_client.UpdateLoginFlowWithOidcMethod)
	parseBody(r, body)

	_, resp, e := kratos.FrontendApi.
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

func handleKratosError(w http.ResponseWriter, _ *http.Request) {
	// kratos := NewKratosClient()
	return
}

func handleConsent(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kratos := NewKratosClient()
	hydra := NewHydraClient()

	// Get the Kratos session to make sure that the user is actually logged in
	session, session_resp, e := kratos.FrontendApi.ToSession(context.Background()).
		Cookie(cookiesToString(r.Cookies())).
		Execute()
	if e != nil {
		log.Printf("Error when calling `FrontendApi.ToSession`: %v\n", e)
		log.Printf("Full HTTP response: %v\n", session_resp)
		return
	}

	ses, ok := session.Identity.Traits.(map[string]interface{})
	if !ok {
		// We should never end up here
		log.Printf("Unexpected traits format: %v\n", ok)
	}
	for k, v := range ses {
		log.Printf("%v == %v\n", k, v)

	}

	// Get the consent request
	consent, consent_resp, e := hydra.AdminApi.GetConsentRequest(context.Background()).
		ConsentChallenge(q.Get("consent_challenge")).
		Execute()
	if e != nil {
		log.Printf("Error when calling `AdminApi.GetConsentRequest`: %v\n", e)
		log.Printf("Full HTTP response: %v\n", consent_resp)
		return
	}

	accept_consent_req := hydra_client.NewAcceptConsentRequest()
	accept_consent_req.SetGrantScope(consent.RequestedScope)
	accept_consent_req.SetGrantAccessTokenAudience(consent.RequestedAccessTokenAudience)
	accept, accept_resp, e := hydra.AdminApi.AcceptConsentRequest(context.Background()).
		ConsentChallenge(q.Get("consent_challenge")).
		AcceptConsentRequest(*accept_consent_req).
		Execute()
	if e != nil {
		log.Printf("Error when calling `AdminApi.AcceptConsentRequest`: %v\n", e)
		log.Printf("Full HTTP response: %v\n", accept_resp)
		return
	}

	resp, e := accept.MarshalJSON()
	if e != nil {
		log.Printf("Error when marshalling Json: %v\n", e)
		return
	}
	w.WriteHeader(200)
	w.Write(resp)

	return
}

func writeResponse(w http.ResponseWriter, r *http.Response) {
	for k, vs := range r.Header {
		for _, v := range vs {
			w.Header().Set(k, v)
		}
	}
	// We need to set the headers before setting the status code, otherwise
	// the response writer freaks out
	w.WriteHeader(r.StatusCode)
	// This assumes POST
	// TODO: Handle other response methods
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
	fmt.Fprint(w, string(body))
}

func cookiesToString(cookies []*http.Cookie) string {
	var ret []string
	ret = make([]string, len(cookies))
	for i, c := range cookies {
		ret[i] = fmt.Sprintf("%s=%s", c.Name, c.Value)
	}
	return strings.Join(ret, "; ")
}

func parseBody(r *http.Request, body interface{}) *interface{} {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(body)
	if err != nil {
		log.Println(err)
	}
	return &body
}
