package stub

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	api "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	"github.com/syndesisio/syndesis/install/operator/pkg/enmasse"
	"github.com/syndesisio/syndesis/install/operator/pkg/syndesis"

	"k8s.io/api/core/v1"
)

func NewHandler(token string) sdk.Handler {
	return &Handler{
		saToken: token,
	}
}

type Handler struct {
	// Fill me
	saToken string
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	logrus.Info("handing ", event.Object.GetObjectKind().GroupVersionKind().String())
	switch o := event.Object.(type) {
	case *api.Syndesis:
		return syndesis.Reconcile(o, event.Deleted)
	case *v1.ConfigMap:
		return enmasse.ReconcileConfigmap(o, event.Deleted)
	case *api.Connection:
		return enmasse.ReconcileConnection(o, event.Deleted, h.saToken)
	}
	return nil
}
