package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"identity_platform_login_ui/health"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	prometheus "identity_platform_login_ui/prometheus"

	hydra_client "github.com/ory/hydra-client-go/v2"
	kratos_client "github.com/ory/kratos-client-go"
)

const defaultPort = "8080"

var oidcScopeMapping = map[string][]string{
	"openid": {"sub"},
	"profile": {
		"name",
		"family_name",
		"given_name",
		"middle_name",
		"nickname",
		"preferred_username",
		"profile",
		"picture",
		"website",
		"gender",
		"birthdate",
		"zoneinfo",
		"locale",
		"updated_at",
	},
	"email":   {"email", "email_verified"},
	"address": {"address"},
	"phone":   {"phone_number", "phone_number_verified"},
}

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

func getBaseURL(r *http.Request) string {
	if url := os.Getenv("BASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf("%s://%s/%s", r.URL.Scheme, r.Host, r.URL.Path)
}

func main() {
	metricsManager := setUpPrometheus()

	dist, _ := fs.Sub(ui, "ui/dist")
	fs := http.FileServer(http.FS(dist))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Add the html suffix if missing
		// This allows us to serve /login.html in the /login URL
		if ext := path.Ext(r.URL.Path); ext == "" && r.URL.Path != "/" {
			r.URL.Path += ".html"
		}
		metricsManager.Middleware(fs.ServeHTTP)(w, r)
	})

	http.HandleFunc("/api/kratos/self-service/login/browser", metricsManager.Middleware(handleCreateFlow))
	http.HandleFunc("/api/kratos/self-service/login/flows", metricsManager.Middleware(handleLoginFlow))
	http.HandleFunc("/api/kratos/self-service/login", metricsManager.Middleware(handleUpdateFlow))
	http.HandleFunc("/api/kratos/self-service/errors", metricsManager.Middleware(handleKratosError))
	http.HandleFunc("/api/consent", metricsManager.Middleware(handleConsent))
	http.HandleFunc(prometheus.PrometheusPath, metricsManager.Middleware(prometheus.PrometheusMetrics))

	port := os.Getenv("PORT")

	if port == "" {
		port = defaultPort
	}

	log.Println("Starting server on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// TODO: Validate response when server error handling is implemented
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
		if session_resp.StatusCode != 401 {
			if e != nil {
				log.Printf("Error when calling `FrontendApi.ToSession`: %v\n", e)
				log.Printf("Full HTTP response: %v\n", session_resp)
			} else {
				accept := hydra_client.NewAcceptOAuth2LoginRequest(session.Identity.Id)
				hydra := NewHydraClient()
				_, resp, e := hydra.OAuth2Api.AcceptOAuth2LoginRequest(context.Background()).
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
	_, resp, e := kratos.FrontendApi.
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
func handleLoginFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kratos := NewKratosClient()

	_, resp, e := kratos.FrontendApi.
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

// TODO: Validate response when server error handling is implemented
func handleKratosError(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	kratos := NewKratosClient()
	_, resp, e := kratos.FrontendApi.GetFlowError(context.Background()).Id(id).Execute()
	if e != nil {
		log.Printf("Error when calling `FrontendApi.GetFlowError`: %v\n", e)
		log.Printf("Full HTTP response: %v\n", resp)
		return
	}
	writeResponse(w, resp)
	return
}

// TODO: Validate response when server error handling is implemented
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

	// Get the consent request
	consent, consent_resp, e := hydra.OAuth2Api.GetOAuth2ConsentRequest(context.Background()).
		ConsentChallenge(q.Get("consent_challenge")).
		Execute()
	if e != nil {
		log.Printf("Error when calling `AdminApi.GetConsentRequest`: %v\n", e)
		log.Printf("Full HTTP response: %v\n", consent_resp)
		return
	}

	consent_session := hydra_client.NewAcceptOAuth2ConsentRequestSession()
	consent_session.SetIdToken(getUserClaims(session.Identity, *consent))
	accept_consent_req := hydra_client.NewAcceptOAuth2ConsentRequest()
	accept_consent_req.SetGrantScope(consent.RequestedScope)
	accept_consent_req.SetGrantAccessTokenAudience(consent.RequestedAccessTokenAudience)
	accept_consent_req.SetSession(*consent_session)
	accept, accept_resp, e := hydra.OAuth2Api.AcceptOAuth2ConsentRequest(context.Background()).
		ConsentChallenge(q.Get("consent_challenge")).
		AcceptOAuth2ConsentRequest(*accept_consent_req).
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

func getUserClaims(i kratos_client.Identity, cr hydra_client.OAuth2ConsentRequest) map[string]interface{} {
	ret := make(map[string]interface{})
	// Export the user claims and filter them based on the requested scopes
	traits, ok := i.Traits.(map[string]interface{})
	if !ok {
		// We should never end up here
		log.Printf("Unexpected traits format: %v\n", ok)
	}
	log.Println(traits)
	for _, s := range cr.RequestedScope {
		cs, ok := oidcScopeMapping[s]
		if !ok {
			continue
		}
		log.Println(cs)
		log.Println(s)
		for _, c := range cs {
			val, ok := traits[c]
			if ok {
				ret[c] = val
			}
		}
	}

	return ret
}

func setUpPrometheus() *prometheus.MetricsManager {
	mm := prometheus.NewMetricsManagerWithPrefix("identity-platform-login-ui-operator", "http", "", "", "")
	mm.RegisterRoutes(
		"/api/kratos/self-service/login/browser",
		"/api/kratos/self-service/login/flows",
		"/api/kratos/self-service/login",
		"/api/kratos/self-service/errors",
		"/api/consent",
		"/consent.html",
		"/error.html",
		"/index.html",
		"/login.html",
		"/",
		"",
		"/oidc_error",
		"/registration",
		prometheus.PrometheusPath,
	)
	return mm
}
