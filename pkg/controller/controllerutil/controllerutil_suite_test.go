package controllerutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestControllerutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controllerutil Suite")
}

var t *envtest.Environment
var cfg *rest.Config
var c client.Client

var _ = BeforeSuite(func() {
	var err error

	t = &envtest.Environment{}

	cfg, err = t.Start()
	Expect(err).NotTo(HaveOccurred())

	c, err = client.New(cfg, client.Options{})
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	t.Stop()
})
