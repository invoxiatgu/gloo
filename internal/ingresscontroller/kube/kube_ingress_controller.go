package kube

import (
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1beta1listers "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/rest"

	"sort"
	"strings"

	"github.com/solo-io/glue/internal/pkg/kube/controller"
	"github.com/solo-io/glue/internal/pkg/kube/upstream"
	"github.com/solo-io/glue/pkg/api/types/v1"
	"github.com/solo-io/glue/pkg/log"
	clientset "github.com/solo-io/glue/pkg/platform/kube/crd/client/clientset/versioned"
	crdv1 "github.com/solo-io/glue/pkg/platform/kube/crd/solo.io/v1"
)

const (
	resourcePrefix    = "glue-generated"
	upstreamPrefix    = resourcePrefix + "-upstream"
	virtualHostPrefix = resourcePrefix + "-virtualhost"

	defaultVirtualHost = "default"

	GlueIngressClass = "glue"
)

type ingressController struct {
	errors             chan error
	useAsGlobalIngress bool

	// where to store generated crds
	crdNamespace string

	ingressLister v1beta1listers.IngressLister
	glueClient    clientset.Interface
}

func (c *ingressController) Error() <-chan error {
	return c.errors
}

func NewIngressController(cfg *rest.Config, resyncDuration time.Duration, stopCh <-chan struct{}, useAsGlobalIngress bool,
	crdNamespace string) (*ingressController, error) {
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kube clientset: %v", err)
	}

	glueClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create glue clientset: %v", err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, resyncDuration)
	ingressInformer := kubeInformerFactory.Extensions().V1beta1().Ingresses()

	c := &ingressController{
		errors:             make(chan error),
		useAsGlobalIngress: useAsGlobalIngress,
		crdNamespace:       crdNamespace,

		ingressLister: ingressInformer.Lister(),
		glueClient:    glueClient,
	}

	kubeController := controller.NewController("glue-ingress-controller", kubeClient,
		c.syncGlueResourcesWithIngresses,
		ingressInformer.Informer())

	go kubeInformerFactory.Start(stopCh)
	go func() {
		kubeController.Run(2, stopCh)
	}()

	return c, nil
}

func (c *ingressController) syncGlueResourcesWithIngresses(namespace, name string, v interface{}) {
	ingress, ok := v.(*v1beta1.Ingress)
	if !ok {
		return
	}
	// only react if it's an ingress we care about
	if !isOurIngress(c.useAsGlobalIngress, ingress) {
		log.Debugf("%v is not our ingress, ignoring", ingress)
		return
	}
	log.Debugf("syncing glue config items after ingress %v/%v changed", namespace, name)
	if err := c.syncGlueResources(); err != nil {
		c.errors <- err
	}
}

func (c *ingressController) syncGlueResources() error {
	desiredUpstreams, desiredVirtualHosts, err := c.generateDesiredCrds()
	if err != nil {
		return fmt.Errorf("failed to generate desired crds: %v", err)
	}
	actualUpstreams, actualVirtualHosts, err := c.getActualCrds()
	if err != nil {
		return fmt.Errorf("failed to list actual crds: %v", err)
	}
	if err := c.syncUpstreams(desiredUpstreams, actualUpstreams); err != nil {
		return fmt.Errorf("failed to sync actual with desired upstreams: %v", err)
	}
	if err := c.syncVirtualHosts(desiredVirtualHosts, actualVirtualHosts); err != nil {
		return fmt.Errorf("failed to sync actual with desired virtualHosts: %v", err)
	}
	return nil
}

func (c *ingressController) getActualCrds() ([]crdv1.Upstream, []crdv1.VirtualHost, error) {
	upstreams, err := c.glueClient.GlueV1().Upstreams(c.crdNamespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get upstream crd list: %v", err)
	}
	virtualHosts, err := c.glueClient.GlueV1().VirtualHosts(c.crdNamespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get virtual host crd list: %v", err)
	}
	return upstreams.Items, virtualHosts.Items, nil
}

func (c *ingressController) generateDesiredCrds() ([]crdv1.Upstream, []crdv1.VirtualHost, error) {
	ingressList, err := c.ingressLister.List(labels.Everything())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list ingresses: %v", err)
	}
	upstreamsByName := make(map[string]v1.Upstream)
	routesByHostName := make(map[string][]v1.Route)
	sslsByHostName := make(map[string]v1.SSLConfig)
	// deterministic way to sort ingresses
	sort.SliceStable(ingressList, func(i, j int) bool {
		return strings.Compare(ingressList[i].Name, ingressList[j].Name) > 0
	})
	for _, ingress := range ingressList {
		// only care if it's our ingress class, or we're the global default
		if !isOurIngress(c.useAsGlobalIngress, ingress) {
			continue
		}
		// configure ssl for each host
		for _, tls := range ingress.Spec.TLS {
			if len(tls.Hosts) == 0 {
				sslsByHostName[defaultVirtualHost] = v1.SSLConfig{SecretRef: tls.SecretName}
			}
			for _, host := range tls.Hosts {
				sslsByHostName[host] = v1.SSLConfig{SecretRef: tls.SecretName}
			}
		}
		// default virtualhost
		if ingress.Spec.Backend != nil {
			us := newUpstreamFromBackend(ingress.Namespace, *ingress.Spec.Backend)
			if _, ok := routesByHostName[defaultVirtualHost]; ok {
				runtime.HandleError(errors.Errorf("default backend was redefined in ingress %v, ignoring", ingress.Name))
			} else {
				routesByHostName[defaultVirtualHost] = []v1.Route{
					{
						Matcher: v1.Matcher{
							Path: v1.Path{
								Prefix: "/",
							},
						},
						Destination: v1.Destination{
							SingleDestination: v1.SingleDestination{
								UpstreamDestination: &v1.UpstreamDestination{
									UpstreamName: us.Name,
								},
							},
						},
					},
				}
			}
		}
		for _, rule := range ingress.Spec.Rules {
			addRoutesAndUpstreams(ingress.Namespace, rule, upstreamsByName, routesByHostName)
		}
	}
	uniqueVirtualHosts := make(map[string]v1.VirtualHost)
	for host, routes := range routesByHostName {
		// sort routes by path length
		// equal length sorted by string compare
		// longest routes should come first
		sortRoutes(routes)
		// TODO: evaluate
		// set default virtualhost to match *
		domains := []string{host}
		if host == defaultVirtualHost {
			domains[0] = "*"
		}
		uniqueVirtualHosts[host] = v1.VirtualHost{
			Name: host,
			// kubernetes only supports a single domain per virtualhost
			Domains:   domains,
			Routes:    routes,
			SSLConfig: sslsByHostName[host],
		}
	}
	var (
		upstreams    []crdv1.Upstream
		virtualHosts []crdv1.VirtualHost
	)
	for _, us := range upstreamsByName {
		upstreams = append(upstreams, crdv1.UpstreamToCRD(metav1.ObjectMeta{
			Name:      us.Name,
			Namespace: c.crdNamespace,
		}, us))
	}
	for name, virtualHost := range uniqueVirtualHosts {
		if name != defaultVirtualHost {
			name = fmt.Sprintf("%s-%s", virtualHostPrefix, name)
		}
		virtualHosts = append(virtualHosts, crdv1.VirtualHostToCRD(metav1.ObjectMeta{
			Name:      name,
			Namespace: c.crdNamespace,
		}, virtualHost))
	}
	return upstreams, virtualHosts, nil
}

func sortRoutes(routes []v1.Route) {
	sort.SliceStable(routes, func(i, j int) bool {
		p1 := routes[i].Matcher.Path.Regex
		p2 := routes[j].Matcher.Path.Regex
		l1 := len(p1)
		l2 := len(p2)
		if l1 == l2 {
			return strings.Compare(p1, p2) < 0
		}
		// longer = comes first
		return l1 > l2
	})
}

func (c *ingressController) syncUpstreams(desiredUpstreams, actualUpstreams []crdv1.Upstream) error {
	var (
		upstreamsToCreate []crdv1.Upstream
		upstreamsToUpdate []crdv1.Upstream
	)
	for _, desiredUpstream := range desiredUpstreams {
		var update bool
		for i, actualUpstream := range actualUpstreams {
			if desiredUpstream.Name == actualUpstream.Name {
				// modify existing upstream
				desiredUpstream.ResourceVersion = actualUpstream.ResourceVersion
				update = true
				if !reflect.DeepEqual(desiredUpstream.Spec, actualUpstream.Spec) {
					// only actually update if the spec has changed
					upstreamsToUpdate = append(upstreamsToUpdate, desiredUpstream)
				}
				// remove it from the list we match against
				actualUpstreams = append(actualUpstreams[:i], actualUpstreams[i+1:]...)
				break
			}
		}
		if !update {
			// desired was not found, mark for creation
			upstreamsToCreate = append(upstreamsToCreate, desiredUpstream)
		}
	}
	for _, us := range upstreamsToCreate {
		if _, err := c.glueClient.GlueV1().Upstreams(c.crdNamespace).Create(&us); err != nil {
			return fmt.Errorf("failed to create upstream crd %s: %v", us.Name, err)
		}
	}
	for _, us := range upstreamsToUpdate {
		if _, err := c.glueClient.GlueV1().Upstreams(c.crdNamespace).Update(&us); err != nil {
			return fmt.Errorf("failed to update upstream crd %s: %v", us.Name, err)
		}
	}
	// only remaining are no longer desired, delete em!
	for _, us := range actualUpstreams {
		if err := c.glueClient.GlueV1().Upstreams(c.crdNamespace).Delete(us.Name, nil); err != nil {
			return fmt.Errorf("failed to update upstream crd %s: %v", us.Name, err)
		}
	}
	return nil
}

func (c *ingressController) syncVirtualHosts(desiredVirtualHosts, actualVirtualHosts []crdv1.VirtualHost) error {
	var (
		virtualHostsToCreate []crdv1.VirtualHost
		virtualHostsToUpdate []crdv1.VirtualHost
	)
	for _, desiredVirtualHost := range desiredVirtualHosts {
		var update bool
		for i, actualVirtualHost := range actualVirtualHosts {
			if desiredVirtualHost.Name == actualVirtualHost.Name {
				// modify existing virtualHost
				desiredVirtualHost.ResourceVersion = actualVirtualHost.ResourceVersion
				update = true
				if !reflect.DeepEqual(desiredVirtualHost.Spec, actualVirtualHost.Spec) {
					// only actually update if the spec has changed
					virtualHostsToUpdate = append(virtualHostsToUpdate, desiredVirtualHost)
				}
				// remove it from the list we match against
				actualVirtualHosts = append(actualVirtualHosts[:i], actualVirtualHosts[i+1:]...)
				break
			}
		}
		if !update {
			// desired was not found, mark for creation
			virtualHostsToCreate = append(virtualHostsToCreate, desiredVirtualHost)
		}
	}
	for _, virtualHost := range virtualHostsToCreate {
		if _, err := c.glueClient.GlueV1().VirtualHosts(c.crdNamespace).Create(&virtualHost); err != nil {
			return fmt.Errorf("failed to create virtualHost crd %s: %v", virtualHost.Name, err)
		}
	}
	for _, virtualHost := range virtualHostsToUpdate {
		if _, err := c.glueClient.GlueV1().VirtualHosts(c.crdNamespace).Update(&virtualHost); err != nil {
			return fmt.Errorf("failed to update upstream crd %s: %v", virtualHost.Name, err)
		}
	}
	// only remaining are no longer desired, delete em!
	for _, virtualHost := range actualVirtualHosts {
		if err := c.glueClient.GlueV1().VirtualHosts(c.crdNamespace).Delete(virtualHost.Name, nil); err != nil {
			return fmt.Errorf("failed to update upstream crd %s: %v", virtualHost.Name, err)
		}
	}
	return nil
}

func addRoutesAndUpstreams(namespace string, rule v1beta1.IngressRule, upstreams map[string]v1.Upstream, routes map[string][]v1.Route) {
	if rule.HTTP == nil {
		return
	}
	for _, path := range rule.HTTP.Paths {
		generatedUpstream := newUpstreamFromBackend(namespace, path.Backend)
		upstreams[generatedUpstream.Name] = generatedUpstream
		host := rule.Host
		if host == "" {
			host = defaultVirtualHost
		}
		routes[rule.Host] = append(routes[rule.Host], v1.Route{
			Matcher: v1.Matcher{
				Path: v1.Path{
					Regex: path.Path,
				},
			},
			Destination: v1.Destination{
				SingleDestination: v1.SingleDestination{
					UpstreamDestination: &v1.UpstreamDestination{
						UpstreamName: generatedUpstream.Name,
					},
				},
			},
		})
	}
}

func newUpstreamFromBackend(namespace string, backend v1beta1.IngressBackend) v1.Upstream {
	return v1.Upstream{
		Name: upstreamName(namespace, backend),
		Type: upstream.Kubernetes,
		Spec: upstream.ToMap(upstream.Spec{
			ServiceName:      backend.ServiceName,
			ServiceNamespace: namespace,
			ServicePortName:  backend.ServicePort.String(),
		}),
	}
}

func upstreamName(namespace string, backend v1beta1.IngressBackend) string {
	return fmt.Sprintf("%s-%s-%s-%s", upstreamPrefix, namespace, backend.ServiceName, backend.ServicePort.String())
}

func isOurIngress(useAsGlobalIngress bool, ingress *v1beta1.Ingress) bool {
	return useAsGlobalIngress || ingress.Annotations["kubernetes.io/ingress.class"] == GlueIngressClass
}
