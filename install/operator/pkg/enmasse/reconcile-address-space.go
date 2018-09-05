package enmasse

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	errors2 "k8s.io/apimachinery/pkg/api/errors"

	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pkg/errors"

	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	enmasse "github.com/syndesisio/syndesis/install/operator/pkg/apis/enmasse/v1alpha1"
	syndesis "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var errExists = fmt.Errorf("user already exists")

// Reconcile the state
func ReconcileConfigmap(configmap *v1.ConfigMap, deleted bool) error {
	// TODO: Get configmap of enmasse standard
	if val, ok := configmap.ObjectMeta.Labels["type"]; ok && val == "address-space" {
		return reconcileConfigMap(configmap, deleted)
	}
	return nil
}

func reconcileConfigMap(configmap *v1.ConfigMap, deleted bool) error {
	addressSpace, err := getAddressSpaceObj(configmap)
	if err != nil {
		logrus.Errorf("failed to get enmasse address space object %+v", err)
		return err
	}

	// TODO: If not there, create it. If exist, return nil
	err = sdk.Get(getConnectionCRObj(addressSpace))

	namespace, _ := k8sutil.GetWatchNamespace()
	conn := syndesis.Connection{ObjectMeta: metav1.ObjectMeta{Name: addressSpace.Name, Namespace: namespace}, TypeMeta: metav1.TypeMeta{APIVersion: "syndesis.io/v1alpha1", Kind: "Connection"}}
	if deleted {
		return sdk.Delete(&conn, sdk.WithDeleteOptions(metav1.NewDeleteOptions(0)))
	}

	logrus.Infoln("Reconciling connection...")
	user, pass, err := getKeycloakAdminCredentials()
	if err != nil {
		return err
	}
	routeClient, err := routev1.NewForConfig(k8sclient.GetKubeConfig())
	if err != nil {
		return err
	}

	route, err := routeClient.Routes(namespace).Get("keycloak", metav1.GetOptions{})
	if err != nil {
		return err
	}
	host := fmt.Sprintf("https://%s", route.Spec.Host)
	realm := addressSpace.Annotations["enmasse.io/realm-name"]

	token, err := keycloakLogin(host, user, pass)
	if err != nil {
		return err
	}
	kcUser, err := createUser(host, token, realm, "testintegration", "password")
	if err == errExists {
		logrus.Info("user already exists doing nothing")

	}
	if err != nil && err != errExists {
		return err
	}

	var msgHost = "amqp://"

	for _, ep := range addressSpace.Status.EndpointStatuses {
		if ep.Name == "messaging" {
			msgHost = fmt.Sprintf("%v%v", msgHost, ep.ServiceHost)
			for _, p := range ep.ServicePorts {
				if p.Name == "amqp" {
					msgHost = fmt.Sprintf("%v:%v", msgHost, p.Port)
				}
			}
		}
	}
	if msgHost == "" {
		return errors.New("failed to get messaging host from address-space")
	}
	msgHost += "?amqp.saslMechanisms=PLAIN"
	conn.Spec = syndesis.ConnectionSpec{Password: kcUser.Password, URL: msgHost, Username: kcUser.UserName}
	if err := sdk.Create(&conn); err != nil {
		if errors2.IsAlreadyExists(err) {
			logrus.Info("connection already exists doing nothing")
		}
		return err
	}
	return nil
}

func getAddressSpaceObj(configmap *v1.ConfigMap) (*enmasse.AddressSpace, error) {
	var addressSpace enmasse.AddressSpace
	if err := json.Unmarshal([]byte(configmap.Data["config.json"]), &addressSpace); err != nil {
		logrus.Errorf("Failed to unmarshal the configmap data into an AddressSpace object %+v", err)
		return nil, err
	}
	return &addressSpace, nil
}

func getConnectionCRObj(addressSpace *enmasse.AddressSpace) *syndesis.Connection {
	var messagingEndpoint enmasse.EndpointStatus
	var amqpPort string

	for _, v := range addressSpace.Status.EndpointStatuses {
		if v.Name == "messaging" {
			messagingEndpoint = v
		}
	}
	for _, v := range messagingEndpoint.ServicePorts {
		if v.Name == "amqp" {
			amqpPort = string(v.Port)
		}
	}
	amqpURL := "amqp://" + messagingEndpoint.ServiceHost + ":" + amqpPort + "?amqp.saslMechanisms=PLAIN"

	return &syndesis.Connection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "connection-" + addressSpace.Name,
			Namespace: addressSpace.Namespace,
		},
		Spec: syndesis.ConnectionSpec{
			URL: amqpURL,
		},
		Status: syndesis.ConnectionStatus{
			Ready: false,
			Phase: "PendingAuth",
		},
	}
}

func defaultRequester() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c := &http.Client{Transport: transport, Timeout: time.Second * 10}
	return c
}

func keycloakLogin(host, user, pass string) (string, error) {
	var authUrl = "auth/realms/master/protocol/openid-connect/token"
	form := url.Values{}
	form.Add("username", user)
	form.Add("password", pass)
	form.Add("client_id", "admin-cli")
	form.Add("grant_type", "password")

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/%s", host, authUrl),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", errors.Wrap(err, "error creating login request")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res, err := defaultRequester().Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return "", errors.Wrap(err, "error performing token request")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logrus.Errorf("error reading response %+v", err)
		return "", errors.Wrap(err, "error reading token response")
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.New("unexpected status code when logging in" + res.Status)
	}
	tokenRes := &tokenResponse{}
	err = json.Unmarshal(body, tokenRes)
	if err != nil {
		logrus.Error("failed to parse body ", string(body), res.StatusCode)
		return "", errors.Wrap(err, "error parsing token response ")
	}

	if tokenRes.Error != "" {
		logrus.Errorf("error with request: " + tokenRes.ErrorDescription)
		return "", errors.New(tokenRes.ErrorDescription)
	}

	return tokenRes.AccessToken, nil

}

func getKeycloakAdminCredentials() (string, string, error) {
	kcCreds := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-credentials",
			Namespace: os.Getenv("WATCH_NAMESPACE"),
		},
	}
	err := sdk.Get(kcCreds)
	if err != nil {
		logrus.Errorf("failed to get keycloak credentials secret %+v", err)
		return "", "", err
	}

	decodedParams := map[string]string{}
	for k, v := range kcCreds.Data {
		decodedParams[k] = string(v)
	}

	return decodedParams["admin.username"], decodedParams["admin.password"], nil
}

func createUser(host, token, realm, userName, password string) (*User, error) {
	//https://keycloak-john-integration.apps.sedroche.openshiftworkshop.com/auth/admin/realms/john-integration-messaging-service/users
	//https://keycloak-john-integration.apps.sedroche.openshiftworkshop.com/auth/admin/realms/John-integration-messaging-service/users
	u := "%s/auth/admin/realms/%s/users"
	url := fmt.Sprintf(u, host, realm)
	logrus.Debug("calling keycloak url: ", url)
	user := &keycloakApiUser{
		UserName: userName,
		Credentials: []credential{
			{
				Type:      "password",
				Value:     password,
				Temporary: false,
			},
		},
		Enabled: true,
		RealmRoles: []string{
			"offline_access",
			"uma_authorization",
		},
		Groups: []string{
			"send_*",
			"recv_*",
			"monitor",
			"manage",
		},
	}
	body, err := json.Marshal(user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request body for creating new enmasse user")
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request to create new user")
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	res, err := defaultRequester().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request to create new enmasse user")
	}
	defer res.Body.Close()
	kcUser := &User{UserName: userName, Password: password}
	if res.StatusCode == http.StatusConflict {
		logrus.Info("user already exists doing nothing")
		return kcUser, errExists

	}
	if res.StatusCode != http.StatusCreated {
		return nil, errors.New("failed to create user status code " + res.Status)
	}
	return kcUser, nil
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type keycloakApiUser struct {
	ID              string              `json:"id,omitempty"`
	UserName        string              `json:"username,omitempty"`
	FirstName       string              `json:"firstName"`
	LastName        string              `json:"lastName"`
	Email           string              `json:"email,omitempty"`
	EmailVerified   bool                `json:"emailVerified"`
	Enabled         bool                `json:"enabled"`
	RealmRoles      []string            `json:"realmRoles,omitempty"`
	ClientRoles     map[string][]string `json:"clientRoles"`
	RequiredActions []string            `json:"requiredActions,omitempty"`
	Groups          []string            `json:"groups,omitempty"`
	Credentials     []credential        `json:"credentials"`
}

type credential struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}

type User struct {
	UserName string
	Password string
}
