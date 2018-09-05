package enmasse

import (
	"os"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	api "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
)

// ReconcileConnection
func ReconcileConnection(connection *api.Connection, deleted bool, token string) error {
	switch connection.Status.Phase {
	case "":
		err := createConnection(connection, os.Getenv("SYNDESIS_SERVER_SERVICE_HOST"), token)
		if err != nil {
			connection.Status.Phase = "failed_creation"
			connection.Status.Ready = false
			sdk.Update(connection)
			return err
		}
		connection.Status.Phase = "ready"
		connection.Status.Ready = true
		return sdk.Update(connection)
	}

	return nil
}
