package main

import (
	"context"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/syndesisio/syndesis/install/operator/pkg/syndesis/legacy"

	// Load Openshift types
	_ "github.com/syndesisio/syndesis/install/operator/pkg/openshift"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/syndesisio/syndesis/install/operator/pkg/stub"

	"flag"

	"github.com/sirupsen/logrus"
	configuration "github.com/syndesisio/syndesis/install/operator/pkg/syndesis/configuration"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operatorof the Syndesis infrastructure elements-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()

	configuration.TemplateLocation = flag.String("template", "/conf/syndesis-template.yml", "Path to template used for installation")
	flag.Parse()
	logrus.Infof("Using template %s", *configuration.TemplateLocation)

	var token = os.Getenv("SA_TOKEN")
	if token == "" {
		//read token from file
		data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
		if err != nil {
			panic(err)
		}
		token = string(data)
	}

	resource := "syndesis.io/v1alpha1"
	kind := "Syndesis"
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("Failed to get watch namespace: %v", err)
	}

	ctx := context.TODO()

	legacyController := legacy.NewLegacyController(namespace)
	legacyController.Start(ctx)

	sdk.Watch(resource, "Connection", namespace, 10*time.Second)
	sdk.Watch(resource, kind, namespace, 10*time.Second)
	if os.Getenv("ENABLE_ENMASSE") != "false" {
		sdk.Watch("v1", "ConfigMap", namespace, 10*time.Second, sdk.WithLabelSelector("type=address-space"))
	}
	sdk.Handle(stub.NewHandler(token))
	sdk.Run(ctx)
}
