package enmasse

import (
	"bytes"
	"fmt"
	"strings"

	"net/http"

	glog "github.com/sirupsen/logrus"

	api "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
)

type apiResponse struct {
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func createConnection(connection *api.Connection, apiHost, token string) error {
	glog.Infof("Creating connection")
	apiHost = "http://" + apiHost + "/api/v1/connections"
	body := getBody(connection.Spec.Username, connection.Spec.Password, connection.Spec.URL, connection.ObjectMeta.Name)
	req, err := http.NewRequest("POST", apiHost, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-FORWARDED-USER", "syndesis-operator")
	req.Header.Set("X-FORWARDED-ACCESS-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("SYNDESIS-XSRF-TOKEN", "awesome")
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return fmt.Errorf("Failed to create connection: %v", rsp.Status)
	}
	return nil
}

func getBody(username, password, url, connectionName string) *bytes.Buffer {
	populatedBody := strings.Replace(body, "<<<CONNECTION_URI>>>", url, 1)
	populatedBody = strings.Replace(populatedBody, "<<<USERNAME>>>", username, 1)
	populatedBody = strings.Replace(populatedBody, "<<<PASSWORD>>>", password, 1)
	populatedBody = strings.Replace(populatedBody, "<<<CONNECTION_NAME>>>", connectionName, 1)
	return bytes.NewBufferString(populatedBody)
}

const body = `{
"connector": {
"description": "Subscribe for and publish messages.",
"icon": "fa-amqp",
"componentScheme": "amqp",
"connectorFactory": "io.syndesis.connector.amqp.AMQPConnectorFactory",
"actionsSummary": {
	"totalActions": 3
},
"uses": 0,
"id": "amqp",
"version": 1,
"actions": [{
		"id": "io.syndesis:amqp-publish-action",
		"name": "Publish messages",
		"description": "Send data to the destination you specify.",
		"descriptor": {
			"inputDataShape": {
				"kind": "any"
			},
			"outputDataShape": {
				"kind": "none"
			},
			"propertyDefinitionSteps": [{
				"description": "Specify AMQP destination properties, including Queue or Topic name",
				"name": "Select the AMQP Destination",
				"properties": {
					"deliveryPersistent": {
						"componentProperty": false,
						"defaultValue": "true",
						"deprecated": false,
						"labelHint": "Message delivery is guaranteed when Persistent is selected.",
						"displayName": "Persistent",
						"group": "producer",
						"javaType": "boolean",
						"kind": "parameter",
						"label": "producer",
						"required": false,
						"secret": false,
						"type": "boolean",
						"order": 3
					},
					"destinationName": {
						"componentProperty": false,
						"deprecated": false,
						"labelHint": "Name of the queue or topic to send data to.",
						"displayName": "Destination Name",
						"group": "common",
						"javaType": "java.lang.String",
						"kind": "path",
						"required": true,
						"secret": false,
						"type": "string",
						"order": 1
					},
					"destinationType": {
						"componentProperty": false,
						"defaultValue": "queue",
						"deprecated": false,
						"labelHint": "By default, the destination is a Queue.",
						"displayName": "Destination Type",
						"group": "common",
						"javaType": "java.lang.String",
						"kind": "path",
						"required": false,
						"secret": false,
						"type": "string",
						"order": 2,
						"enum": [{
								"label": "Topic",
								"value": "topic"
							},
							{
								"label": "Queue",
								"value": "queue"
							}
						]
					}
				}
			}]
		},
		"actionType": "connector",
		"pattern": "To"
	},
	{
		"id": "io.syndesis:amqp-subscribe-action",
		"name": "Subscribe for messages",
		"description": "Receive data from the destination you specify.",
		"descriptor": {
			"inputDataShape": {
				"kind": "none"
			},
			"outputDataShape": {
				"kind": "any"
			},
			"propertyDefinitionSteps": [{
				"description": "Specify AMQP destination properties, including Queue or Topic Name",
				"name": "Select the AMQP Destination",
				"properties": {
					"destinationName": {
						"componentProperty": false,
						"deprecated": false,
						"labelHint": "Name of the queue or topic to receive data from.",
						"displayName": "Destination Name",
						"group": "common",
						"javaType": "java.lang.String",
						"kind": "path",
						"required": true,
						"secret": false,
						"type": "string",
						"order": 1
					},
					"destinationType": {
						"componentProperty": false,
						"defaultValue": "queue",
						"deprecated": false,
						"labelHint": "By default, the destination is a Queue.",
						"displayName": "Destination Type",
						"group": "common",
						"javaType": "java.lang.String",
						"kind": "path",
						"required": false,
						"secret": false,
						"type": "string",
						"order": 2,
						"enum": [{
							"label": "Topic",
							"value": "topic"
						}, {
							"label": "Queue",
							"value": "queue"
						}]
					},
					"durableSubscriptionId": {
						"componentProperty": false,
						"deprecated": false,
						"labelHint": "Set the ID that lets connections close and reopen with missing messages. Connection type must be a topic.",
						"displayName": "Durable Subscription ID",
						"group": "consumer",
						"javaType": "java.lang.String",
						"kind": "parameter",
						"label": "consumer",
						"required": false,
						"secret": false,
						"type": "string",
						"order": 3
					},
					"messageSelector": {
						"componentProperty": false,
						"deprecated": false,
						"labelHint": "Specify a filter expression to receive only data that meets certain criteria.",
						"displayName": "Message Selector",
						"group": "consumer (advanced)",
						"javaType": "java.lang.String",
						"kind": "parameter",
						"label": "consumer,advanced",
						"required": false,
						"secret": false,
						"type": "string",
						"order": 4
					}
				}
			}]
		},
		"actionType": "connector",
		"pattern": "From"
	}, {
		"id": "io.syndesis:amqp-request-action",
		"name": "Request response using messages",
		"description": "Send data to the destination you specify and receive a response.",
		"descriptor": {
			"inputDataShape": {
				"kind": "any"
			},
			"outputDataShape": {
				"kind": "any"
			},
			"propertyDefinitionSteps": [{
				"description": "Specify AMQP destination properties, including Queue or Topic Name",
				"name": "Select the AMQP Destination",
				"properties": {
					"destinationName": {
						"componentProperty": false,
						"deprecated": false,
						"labelHint": "Name of the queue or topic to receive data from.",
						"displayName": "Destination Name",
						"group": "common",
						"javaType": "java.lang.String",
						"kind": "path",
						"required": true,
						"secret": false,
						"type": "string",
						"order": 1
					},
					"destinationType": {
						"componentProperty": false,
						"defaultValue": "queue",
						"deprecated": false,
						"labelHint": "By default, the destination is a Queue.",
						"displayName": "Destination Type",
						"group": "common",
						"javaType": "java.lang.String",
						"kind": "path",
						"required": false,
						"secret": false,
						"type": "string",
						"order": 2,
						"enum": [{
							"label": "Topic",
							"value": "topic"
						}, {
							"label": "Queue",
							"value": "queue"
						}]
					},
					"durableSubscriptionId": {
						"componentProperty": false,
						"deprecated": false,
						"labelHint": "Set the ID that lets connections close and reopen with missing messages. Connection type must be a topic.",
						"displayName": "Durable Subscription ID",
						"group": "consumer",
						"javaType": "java.lang.String",
						"kind": "parameter",
						"label": "consumer",
						"required": false,
						"secret": false,
						"type": "string",
						"order": 3
					},
					"messageSelector": {
						"componentProperty": false,
						"deprecated": false,
						"labelHint": "Specify a filter expression to receive only data that meets certain criteria.",
						"displayName": "Message Selector",
						"group": "consumer (advanced)",
						"javaType": "java.lang.String",
						"kind": "parameter",
						"label": "consumer,advanced",
						"required": false,
						"secret": false,
						"type": "string",
						"order": 4
					}
				}
			}]
		},
		"actionType": "connector",
		"pattern": "To"
	}
],
"tags": ["verifier"],
"name": "AMQP Message Broker",
"properties": {
	"brokerCertificate": {
		"componentProperty": true,
		"deprecated": false,
		"description": "AMQ Broker X.509 PEM Certificate",
		"displayName": "Broker Certificate",
		"group": "security",
		"javaType": "java.lang.String",
		"kind": "property",
		"label": "common,security",
		"required": false,
		"secret": false,
		"type": "textarea",
		"relation": [{
			"action": "ENABLE",
			"when": [{
				"id": "skipCertificateCheck",
				"value": "false"
			}]
		}],
		"order": 6
	},
	"clientCertificate": {
		"componentProperty": true,
		"deprecated": false,
		"description": "AMQ Client X.509 PEM Certificate",
		"displayName": "Client Certificate",
		"group": "security",
		"javaType": "java.lang.String",
		"kind": "property",
		"label": "common,security",
		"required": false,
		"secret": false,
		"type": "textarea",
		"relation": [{
			"action": "ENABLE",
			"when": [{
				"id": "skipCertificateCheck",
				"value": "false"
			}]
		}],
		"order": 7
	},
	"clientID": {
		"componentProperty": true,
		"deprecated": false,
		"labelHint": "Required for connections to close and reopen without missing messages. Connection destination must be a topic.",
		"displayName": "Client ID",
		"group": "security",
		"javaType": "java.lang.String",
		"kind": "property",
		"label": "common,security",
		"required": false,
		"secret": false,
		"type": "string",
		"order": 4
	},
	"connectionUri": {
		"componentProperty": true,
		"deprecated": false,
		"labelHint": "Location to send data to or obtain data from.",
		"displayName": "Connection URI",
		"group": "common",
		"javaType": "java.lang.String",
		"kind": "property",
		"label": "common",
		"required": true,
		"secret": false,
		"type": "string",
		"order": 1
	},
	"password": {
		"componentProperty": true,
		"deprecated": false,
		"labelHint": "Password for the specified user account.",
		"displayName": "Password",
		"group": "security",
		"javaType": "java.lang.String",
		"kind": "property",
		"label": "common,security",
		"required": false,
		"secret": true,
		"type": "string",
		"order": 3
	},
	"skipCertificateCheck": {
		"componentProperty": true,
		"defaultValue": "false",
		"deprecated": false,
		"labelHint": "Ensure certificate checks are enabled for secure production environments. Disable for convenience in only development environments.",
		"displayName": "Check Certificates",
		"group": "security",
		"javaType": "java.lang.String",
		"kind": "property",
		"label": "common,security",
		"required": false,
		"secret": false,
		"type": "string",
		"order": 5,
		"enum": [{
			"label": "Disable",
			"value": "true"
		}, {
			"label": "Enable",
			"value": "false"
		}]
	},
	"username": {
		"componentProperty": true,
		"deprecated": false,
		"labelHint": "Access the broker with this user\u2019s authorization credentials.",
		"displayName": "User Name",
		"group": "security",
		"javaType": "java.lang.String",
		"kind": "property",
		"label": "common,security",
		"required": false,
		"secret": false,
		"type": "string",
		"order": 2
	}
},
"dependencies": [{
	"type": "MAVEN",
	"id": "io.syndesis.connector:connector-amqp:1.5-SNAPSHOT"
}]
},
"icon": "fa-amqp",
"connectorId": "amqp",
"configuredProperties": {
"connectionUri": "<<<CONNECTION_URI>>>",
"username": "<<<USERNAME>>>",
"password": "<<<PASSWORD>>>",
"skipCertificateCheck": "true",
"brokerCertificate": ""
},
"name": "<<<CONNECTION_NAME>>>",
"description": null
}`
