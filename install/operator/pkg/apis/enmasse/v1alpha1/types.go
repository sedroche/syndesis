package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AddressSpaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AddressSpace `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AddressSpace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              AddressSpaceSpec   `json:"spec"`
	Status            AddressSpaceStatus `json:"status,omitempty"`
}

type AddressSpaceSpec struct {
	Type                  string                `json:"type"`
	Plan                  string                `json:"plan"`
	Endpoints             []Endpoint            `json:"endpoints"`
	AuthenticationService AuthenticationService `json:"authenticationService"`
}

type AddressSpaceStatus struct {
	IsReady          bool             `json:"isReady,omitempty"`
	EndpointStatuses []EndpointStatus `json:"endpointStatuses,omitempty"`
}

type AuthenticationService struct {
	Type    string `json:"type,omitempty"`
	Details Detail `json:"details,omitempty"`
}

type Cert struct {
	Provider   string `json:"provider,omitempty"`
	SecretName string `json:"secretName,omitempty"`
}

type Detail struct {
}

type Endpoint struct {
	Name        string `json:"name"`
	Service     string `json:"service,omitempty"`
	ServicePort string `json:"servicePort,omitempty"`
	Cert        Cert   `json:"cert,omitempty"`
}

type EndpointStatus struct {
	Name         string        `json:"name"`
	ServiceHost  string        `json:"serviceHost,omitempty"`
	ServicePorts []ServicePort `json:"servicePorts,omitempty"`
	Host         string        `json:"host,omitempty"`
	Port         int           `json:"port,omitempty"`
}

type ServicePort struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}
