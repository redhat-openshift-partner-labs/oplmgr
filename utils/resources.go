package utils

import (
	"io"

	"github.com/jonboulle/clockwork"

	"k8s.io/cli-runtime/pkg/printers"
	kresource "k8s.io/cli-runtime/pkg/resource"
	kcmdapply "k8s.io/kubectl/pkg/cmd/apply"

	"io/ioutil"

	configapi "k8s.io/client-go/tools/clientcmd/api"

	"context"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"path/filepath"
	"regexp"
	"strings"

	"k8s.io/client-go/discovery/cached/disk"

	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"bytes"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kcmdpatch "k8s.io/kubectl/pkg/cmd/patch"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"reflect"
	"unsafe"

	json "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
)

// https://github.com/openshift/hive/blob/master/pkg/resource/apply.go
// ApplyResult indicates what type of change was performed
// by calling the Apply function
type ApplyResult string

const (
	// ConfiguredApplyResult is returned when a patch was submitted
	ConfiguredApplyResult ApplyResult = "configured"

	// UnchangedApplyResult is returned when no change occurred
	UnchangedApplyResult ApplyResult = "unchanged"

	// CreatedApplyResult is returned when a resource was created
	CreatedApplyResult ApplyResult = "created"

	// UnknownApplyResult is returned when the resulting action could not be determined
	UnknownApplyResult ApplyResult = "unknown"
)

const fieldTooLong metav1.CauseType = "FieldValueTooLong"

// Apply applies the given resource bytes to the target cluster specified by kubeconfig
func (r *helper) Apply(obj []byte) (ApplyResult, error) {
	factory, err := r.getFactory("")
	if err != nil {
		r.logger.WithError(err).Error("failed to obtain factory for apply")
		return "", err
	}
	ioStreams := genericclioptions.IOStreams{
		In:     &bytes.Buffer{},
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
	applyOptions, changeTracker, err := r.setupApplyCommand(factory, obj, ioStreams)
	if err != nil {
		r.logger.WithError(err).Error("failed to setup apply command")
		return "", err
	}

	err = applyOptions.Run()
	if err != nil {
		r.logger.WithError(err).
			WithField("stdout", ioStreams.Out.(*bytes.Buffer).String()).
			WithField("stderr", ioStreams.ErrOut.(*bytes.Buffer).String()).Warn("running the apply command failed")
		return "", err
	}
	return changeTracker.GetResult(), nil
}

// ApplyRuntimeObject serializes an object and applies it to the target cluster specified by the kubeconfig.
func (r *helper) ApplyRuntimeObject(obj runtime.Object, scheme *runtime.Scheme) (ApplyResult, error) {
	data, err := Serialize(obj, scheme)
	if err != nil {
		r.logger.WithError(err).Warn("cannot serialize runtime object")
		return "", err
	}
	return r.Apply(data)
}

func (r *helper) CreateOrUpdate(obj []byte) (ApplyResult, error) {
	factory, err := r.getFactory("")
	if err != nil {
		r.logger.WithError(err).Error("failed to obtain factory for apply")
		return "", err
	}

	errOut := &bytes.Buffer{}
	result, err := r.createOrUpdate(factory, obj, errOut)
	if err != nil {
		r.logger.WithError(err).
			WithField("stderr", errOut.String()).Warn("running the apply command failed")
		return "", err
	}
	return result, nil
}

func (r *helper) CreateOrUpdateRuntimeObject(obj runtime.Object, scheme *runtime.Scheme) (ApplyResult, error) {
	data, err := Serialize(obj, scheme)
	if err != nil {
		r.logger.WithError(err).Warn("cannot serialize runtime object")
		return "", err
	}
	return r.CreateOrUpdate(data)
}

func (r *helper) Create(obj []byte) (ApplyResult, error) {
	factory, err := r.getFactory("")
	if err != nil {
		r.logger.WithError(err).Error("failed to obtain factory for apply")
		return "", err
	}
	result, err := r.createOnly(factory, obj)
	if err != nil {
		r.logger.WithError(err).Warn("running the create command failed")
		return "", err
	}
	return result, nil
}

func (r *helper) CreateRuntimeObject(obj runtime.Object, scheme *runtime.Scheme) (ApplyResult, error) {
	data, err := Serialize(obj, scheme)
	if err != nil {
		r.logger.WithError(err).Warn("cannot serialize runtime object")
		return "", err
	}
	return r.Create(data)
}

func (r *helper) createOnly(f cmdutil.Factory, obj []byte) (ApplyResult, error) {
	info, err := r.getResourceInternalInfo(f, obj)
	if err != nil {
		return "", err
	}
	if info == nil {
		r.logger.Debug("err getting info")
	}
	c, err := f.DynamicClient()
	if err != nil {
		return "", err
	}
	if err = info.Get(); err != nil {
		if !apierrors.IsNotFound(err) {
			return "", err
		}
		// Object doesn't exist yet, create it
		gvr := info.ResourceMapping().Resource
		_, err := c.Resource(gvr).Namespace(info.Namespace).Create(context.TODO(), info.Object.(*unstructured.Unstructured), metav1.CreateOptions{})
		if err != nil {
			return "", err
		}
		return CreatedApplyResult, nil
	}
	return UnchangedApplyResult, nil
}

func (r *helper) createOrUpdate(f cmdutil.Factory, obj []byte, errOut io.Writer) (ApplyResult, error) {
	info, err := r.getResourceInternalInfo(f, obj)
	if err != nil {
		return "", err
	}
	c, err := f.DynamicClient()
	if err != nil {
		return "", err
	}
	sourceObj := info.Object.DeepCopyObject()
	if err = info.Get(); err != nil {
		if !apierrors.IsNotFound(err) {
			return "", err
		}
		// Object doesn't exist yet, create it
		gvr := info.ResourceMapping().Resource
		_, err := c.Resource(gvr).Namespace(info.Namespace).Create(context.TODO(), info.Object.(*unstructured.Unstructured), metav1.CreateOptions{})
		if err != nil {
			return "", err
		}
		return CreatedApplyResult, nil
	}
	openAPISchema, _ := f.OpenAPISchema()
	patcher := kcmdapply.Patcher{
		Mapping:       info.Mapping,
		Helper:        kresource.NewHelper(info.Client, info.Mapping),
		Overwrite:     true,
		BackOff:       clockwork.NewRealClock(),
		OpenapiSchema: openAPISchema,
	}
	sourceBytes, err := runtime.Encode(unstructured.UnstructuredJSONScheme, sourceObj)
	if err != nil {
		return "", err
	}
	patch, _, err := patcher.Patch(info.Object, sourceBytes, info.Source, info.Namespace, info.Name, errOut)
	if err != nil {
		return "", err
	}
	result := ConfiguredApplyResult
	if string(patch) == "{}" {
		result = UnchangedApplyResult
	}
	return result, nil
}

func (r *helper) setupApplyCommand(f cmdutil.Factory, obj []byte, ioStreams genericclioptions.IOStreams) (*kcmdapply.ApplyOptions, *changeTracker, error) {
	r.logger.Debug("setting up apply command")
	o := kcmdapply.NewApplyOptions(ioStreams)
	dynamicClient, err := f.DynamicClient()
	if err != nil {
		r.logger.WithError(err).Error("cannot obtain dynamic client from factory")
		return nil, nil, err
	}
	o.DeleteOptions, err = o.DeleteFlags.ToOptions(dynamicClient, o.IOStreams)
	if err != nil {
		r.logger.WithError(err).Error("cannot create delete options")
		return nil, nil, err
	}
	// Re-use the openAPISchema that should have been initialized in the constructor.
	o.OpenAPISchema = r.openAPISchema
	o.Validator, err = f.Validator(false)
	if err != nil {
		r.logger.WithError(err).Error("cannot obtain schema to validate objects from factory")
		return nil, nil, err
	}
	o.Builder = f.NewBuilder()
	o.Mapper, err = f.ToRESTMapper()
	if err != nil {
		r.logger.WithError(err).Error("cannot obtain RESTMapper from factory")
		return nil, nil, err
	}

	o.DynamicClient = dynamicClient
	o.Namespace, o.EnforceNamespace, err = f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		r.logger.WithError(err).Error("cannot obtain namespace from factory")
		return nil, nil, err
	}
	tracker := &changeTracker{
		internalToPrinter: func(string) (printers.ResourcePrinter, error) { return o.PrintFlags.ToPrinter() },
	}
	o.ToPrinter = tracker.ToPrinter
	info, err := r.getResourceInternalInfo(f, obj)
	if err != nil {
		return nil, nil, err
	}
	o.SetObjects([]*kresource.Info{info})
	return o, tracker, nil
}

type trackerPrinter struct {
	setResult       func()
	internalPrinter printers.ResourcePrinter
}

func (p *trackerPrinter) PrintObj(o runtime.Object, w io.Writer) error {
	if p.setResult != nil {
		p.setResult()
	}
	return p.internalPrinter.PrintObj(o, w)
}

type changeTracker struct {
	result            []ApplyResult
	internalToPrinter func(string) (printers.ResourcePrinter, error)
}

func (t *changeTracker) GetResult() ApplyResult {
	if len(t.result) == 1 {
		return t.result[0]
	}
	return UnknownApplyResult
}

func (t *changeTracker) ToPrinter(name string) (printers.ResourcePrinter, error) {
	var f func()
	switch name {
	case "created":
		f = func() { t.result = append(t.result, CreatedApplyResult) }
	case "configured":
		f = func() { t.result = append(t.result, ConfiguredApplyResult) }
	case "unchanged":
		f = func() { t.result = append(t.result, UnchangedApplyResult) }
	}
	p, err := t.internalToPrinter(name)
	if err != nil {
		return nil, err
	}
	return &trackerPrinter{
		internalPrinter: p,
		setResult:       f,
	}, nil
}

// https://github.com/openshift/hive/blob/master/pkg/resource/client.go
const (
	tokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// GenerateClientConfigFromRESTConfig generates a new kubeconfig using a given rest.Config.
// The rest.Config may come from in-cluster config (as in a pod) or an existing kubeconfig.
func GenerateClientConfigFromRESTConfig(name string, restConfig *rest.Config) *configapi.Config {
	cfg := &configapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       map[string]*configapi.Cluster{},
		AuthInfos:      map[string]*configapi.AuthInfo{},
		Contexts:       map[string]*configapi.Context{},
		CurrentContext: name,
	}

	cluster := &configapi.Cluster{
		Server:                   restConfig.Host,
		InsecureSkipTLSVerify:    restConfig.Insecure,
		CertificateAuthority:     restConfig.CAFile,
		CertificateAuthorityData: restConfig.CAData,
	}

	authInfo := &configapi.AuthInfo{
		ClientCertificate:     restConfig.CertFile,
		ClientCertificateData: restConfig.CertData,
		ClientKey:             restConfig.KeyFile,
		ClientKeyData:         restConfig.KeyData,
		Token:                 restConfig.BearerToken,
		Username:              restConfig.Username,
		Password:              restConfig.Password,
	}

	if restConfig.WrapTransport != nil && len(restConfig.BearerToken) == 0 {
		token, err := ioutil.ReadFile(tokenFile)
		if err != nil {
			log.WithError(err).Warning("empty bearer token and cannot read token file")
		} else {
			authInfo.Token = string(token)
		}
	}

	context := &configapi.Context{
		Cluster:  name,
		AuthInfo: name,
	}

	cfg.Clusters[name] = cluster
	cfg.AuthInfos[name] = authInfo
	cfg.Contexts[name] = context

	return cfg
}

// https://github.com/openshift/hive/blob/master/pkg/resource/delete.go
// DeleteAnyExistingObject will look for any object that exists that matches the passed in 'obj' and will delete it if it exists
func DeleteAnyExistingObject(c client.Client, key client.ObjectKey, obj hivev1.MetaRuntimeObject, logger log.FieldLogger) error {
	logger = logger.WithField("object", key)
	switch err := c.Get(context.Background(), key, obj); {
	case apierrors.IsNotFound(err):
		logger.Debug("object does not exist")
		return nil
	case err != nil:
		logger.WithError(err).Error("error getting object")
		return errors.Wrap(err, "error getting object")
	}
	if obj.GetDeletionTimestamp() != nil {
		logger.Debug("object has already been deleted")
		return nil
	}
	logger.Info("deleting existing object")
	if err := c.Delete(context.Background(), obj); err != nil {
		logger.WithError(err).Error("error deleting object")
		return errors.Wrap(err, "error deleting object")
	}
	return nil
}

func (r *helper) Delete(apiVersion, kind, namespace, name string) error {
	f, err := r.getFactory(namespace)
	if err != nil {
		return errors.Wrap(err, "could not get factory")
	}
	mapper, err := f.ToRESTMapper()
	if err != nil {
		return errors.Wrap(err, "could not get mapper")
	}
	gvk := schema.FromAPIVersionAndKind(apiVersion, kind)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return errors.Wrap(err, "could not get mapping")
	}
	dynamicClient, err := f.DynamicClient()
	if err != nil {
		return errors.Wrap(err, "could not create dynamic client")
	}
	switch err := dynamicClient.Resource(mapping.Resource).Namespace(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); {
	case apierrors.IsNotFound(err):
		r.logger.Info("resource has already been deleted")
	case err != nil:
		return errors.Wrap(err, "could not delete resource")
	}
	return nil
}

// https://github.com/openshift/hive/blob/master/pkg/resource/factory_discovery.go
func getDiscoveryClient(config *rest.Config, cacheDir string) (discovery.CachedDiscoveryInterface, error) {
	config.Burst = 100
	httpCacheDir := filepath.Join(cacheDir, ".kube", "http-cache")
	discoveryCacheDir := computeDiscoverCacheDir(filepath.Join(cacheDir, ".kube", "cache", "discovery"), config.Host)
	return disk.NewCachedDiscoveryClientForConfig(config, discoveryCacheDir, httpCacheDir, time.Duration(10*time.Minute))
}

// overlyCautiousIllegalFileCharacters matches characters that *might* not be supported.  Windows is really restrictive, so this is really restrictive
var overlyCautiousIllegalFileCharacters = regexp.MustCompile(`[^(\w/\.)]`)

// computeDiscoverCacheDir takes the parentDir and the host and comes up with a "usually non-colliding" name.
func computeDiscoverCacheDir(parentDir, host string) string {
	// strip the optional scheme from host if its there:
	schemelessHost := strings.Replace(strings.Replace(host, "https://", "", 1), "http://", "", 1)
	// now do a simple collapse of non-AZ09 characters.  Collisions are possible but unlikely.  Even if we do collide the problem is short lived
	safeHost := overlyCautiousIllegalFileCharacters.ReplaceAllString(schemelessHost, "_")
	return filepath.Join(parentDir, safeHost)
}

// https://github.com/openshift/hive/blob/master/pkg/resource/fake.go
// fakeHelper is a dummy implementation of the resource Helper that will never attempt to communicate with the server.
// Used when communicating with a cluster that is flagged as fake for simulated scale testing.
type fakeHelper struct {
	logger log.FieldLogger
}

// NewFakeHelper returns a new fake helper object that does not actually communicate with the cluster.
func NewFakeHelper(logger log.FieldLogger) Helper {
	r := &fakeHelper{
		logger: logger,
	}
	return r
}

func (r *fakeHelper) Apply(obj []byte) (ApplyResult, error) {
	// TODO: would be good to simulate some of the serialization here if possible so we hit CPU/RAM nearly as much as
	// we would in the real world.
	r.fakeApplySleep()
	return ConfiguredApplyResult, nil
}

func (r *fakeHelper) ApplyRuntimeObject(obj runtime.Object, scheme *runtime.Scheme) (ApplyResult, error) {
	r.fakeApplySleep()
	return ConfiguredApplyResult, nil
}

func (r *fakeHelper) fakeApplySleep() {
	// real world data indicates that for our slowest non-delete request type (POST):
	// histogram_quantile(0.9, (sum without(controller,endpoint,instance,job,namespace,pod,resource,service,status)(rate(hive_kube_client_request_seconds_bucket{remote="true",controller="clustersync"}[2h]))))
	//
	// 50% of requests are under 0.027s
	// 80% of requests are under 0.045s
	// 90% of requests are under 0.053s
	// 99% of requests are under 0.230s
	in := []int{27, 27, 27, 27, 27, 45, 45, 45, 53, 230} // milliseconds
	randomIndex := rand.Intn(len(in))
	wait := time.Duration(in[randomIndex] * 1000000) // nanoseconds to match the duration unit
	r.logger.WithField("sleepMillis", wait.Milliseconds()).Debug("sleeping to simulate an apply")
	time.Sleep(wait)
}

func (r *fakeHelper) CreateOrUpdate(obj []byte) (ApplyResult, error) {
	return ConfiguredApplyResult, nil
}

func (r *fakeHelper) CreateOrUpdateRuntimeObject(obj runtime.Object, scheme *runtime.Scheme) (ApplyResult, error) {
	return ConfiguredApplyResult, nil
}

func (r *fakeHelper) Create(obj []byte) (ApplyResult, error) {
	return ConfiguredApplyResult, nil
}

func (r *fakeHelper) CreateRuntimeObject(obj runtime.Object, scheme *runtime.Scheme) (ApplyResult, error) {
	return ConfiguredApplyResult, nil
}

func (r *fakeHelper) Info(obj []byte) (*Info, error) {
	// TODO: Do we need to fake this better?
	return &Info{}, nil
}

func (fakeHelper) Patch(name types.NamespacedName, kind, apiVersion string, patch []byte, patchType string) error {
	return nil
}

func (fakeHelper) Delete(apiVersion, kind, namespace, name string) error {
	return nil
}

// https://github.com/openshift/hive/blob/master/pkg/resource/info.go
// Info contains information obtained from a resource submitted to the Apply function
type Info struct {
	Name       string
	Namespace  string
	APIVersion string
	Kind       string
	Resource   string
	Object     *unstructured.Unstructured
}

// Info determines the name/namespace and type of the passed in resource bytes
func (r *helper) Info(obj []byte) (*Info, error) {
	factory, err := r.getFactory("")
	if err != nil {
		return nil, err
	}
	resourceInfo, err := r.getResourceInfo(factory, obj)
	if err != nil {
		return nil, err
	}
	return resourceInfo, err
}

func (r *helper) getResourceInternalInfo(f cmdutil.Factory, obj []byte) (*resource.Info, error) {
	builder := f.NewBuilder()
	infos, err := builder.Unstructured().Stream(bytes.NewBuffer(obj), "object").Flatten().Do().Infos()
	if err != nil {
		r.logger.WithError(err).Error("Failed to obtain resource info")
		return nil, fmt.Errorf("could not get info from passed resource: %v", err)
	}
	if len(infos) != 1 {
		r.logger.WithError(err).WithField("infos", infos).Errorf("Expected to get 1 resource info, got %d", len(infos))
		return nil, fmt.Errorf("unexpected number of resources found: %d", len(infos))
	}
	return infos[0], nil
}

func (r *helper) getResourceInfo(f cmdutil.Factory, obj []byte) (*Info, error) {
	info, err := r.getResourceInternalInfo(f, obj)
	if err != nil {
		return nil, err
	}
	return &Info{
		Name:       info.Name,
		Namespace:  info.Namespace,
		Kind:       info.ResourceMapping().GroupVersionKind.Kind,
		APIVersion: info.ResourceMapping().GroupVersionKind.GroupVersion().String(),
		Resource:   info.ResourceMapping().Resource.Resource,
		Object:     info.Object.(*unstructured.Unstructured),
	}, nil
}

// https://github.com/openshift/hive/blob/master/pkg/resource/kubeconfig_factory.go
func (r *helper) getKubeconfigFactory(namespace string) (cmdutil.Factory, error) {
	config, err := clientcmd.Load(r.kubeconfig)
	if err != nil {
		r.logger.WithError(err).Error("an error occurred loading the kubeconfig")
		return nil, err
	}
	overrides := &clientcmd.ConfigOverrides{}
	if len(namespace) > 0 {
		overrides.Context.Namespace = namespace
	}
	clientConfig := clientcmd.NewNonInteractiveClientConfig(*config, "", overrides, nil)
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	if r.metricsEnabled {
		AddControllerMetricsTransportWrapper(restConfig, r.controllerName, r.remote)
	}

	r.logger.WithField("cache-dir", r.cacheDir).Debug("creating cmdutil.Factory from client config and cache directory")
	f := cmdutil.NewFactory(&kubeconfigClientGetter{
		clientConfig:   clientConfig,
		cacheDir:       r.cacheDir,
		controllerName: r.controllerName,
		metricsEnabled: r.metricsEnabled,
		restConfig:     restConfig,
	})
	return f, nil
}

type kubeconfigClientGetter struct {
	clientConfig   clientcmd.ClientConfig
	cacheDir       string
	controllerName hivev1.ControllerName
	metricsEnabled bool
	restConfig     *rest.Config
}

// ToRESTConfig returns restconfig
func (r *kubeconfigClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.restConfig, nil
}

// ToDiscoveryClient returns discovery client
func (r *kubeconfigClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := r.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return getDiscoveryClient(config, r.cacheDir)
}

// ToRESTMapper returns a restmapper
func (r *kubeconfigClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient)
	return expander, nil
}

// ToRawKubeConfigLoader return kubeconfig loader as-is
func (r *kubeconfigClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return r.clientConfig
}

// https://github.com/openshift/hive/blob/master/pkg/resource/patch.go
var (
	patchTypes = map[string]types.PatchType{
		"json":      types.JSONPatchType,
		"merge":     types.MergePatchType,
		"strategic": types.StrategicMergePatchType,
	}
)

// Patch invokes the kubectl patch command with the given resource, patch and patch type
func (r *helper) Patch(name types.NamespacedName, kind, apiVersion string, patch []byte, patchType string) error {

	ioStreams := genericclioptions.IOStreams{
		In:     &bytes.Buffer{},
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
	factory, err := r.getFactory(name.Namespace)
	if err != nil {
		return err
	}
	patchOptions, err := r.setupPatchCommand(name.Name, kind, apiVersion, patchType, factory, string(patch), ioStreams)
	if err != nil {
		r.logger.WithError(err).Error("failed to setup patch command")
		return err
	}
	err = patchOptions.RunPatch()
	if err != nil {
		r.logger.WithError(err).
			WithField("stdout", ioStreams.Out.(*bytes.Buffer).String()).
			WithField("stderr", ioStreams.ErrOut.(*bytes.Buffer).String()).Warn("running the patch command failed")
		return err
	}
	r.logger.
		WithField("stdout", ioStreams.Out.(*bytes.Buffer).String()).
		WithField("stderr", ioStreams.ErrOut.(*bytes.Buffer).String()).Info("patch command successful")
	return nil
}

func (r *helper) setupPatchCommand(name, kind, apiVersion, patchType string, f cmdutil.Factory, patch string, ioStreams genericclioptions.IOStreams) (*kcmdpatch.PatchOptions, error) {

	cmd := kcmdpatch.NewCmdPatch(f, ioStreams)
	cmd.Flags().Parse([]string{})

	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		r.logger.WithError(err).WithField("groupVersion", apiVersion).Error("cannot parse group version")
		return nil, err
	}
	args := []string{fmt.Sprintf("%s.%s.%s/%s", kind, gv.Version, gv.Group, name)}

	o := kcmdpatch.NewPatchOptions(ioStreams)
	o.Complete(f, cmd, args)
	if patchType == "" {
		patchType = "strategic"
	}
	_, ok := patchTypes[patchType]
	if !ok {
		return nil, fmt.Errorf("Invalid patch type: %s. Valid patch types are 'strategic', 'merge' or 'json'", patchType)
	}
	o.PatchType = patchType
	o.Patch = patch

	return o, nil
}

// https://github.com/openshift/hive/blob/master/pkg/resource/restconfig_factory.go
func (r *helper) getRESTConfigFactory(namespace string) (cmdutil.Factory, error) {
	if r.metricsEnabled {
		// Copy the possibly shared restConfig reference and add a metrics wrapper.
		cfg := rest.CopyConfig(r.restConfig)
		AddControllerMetricsTransportWrapper(cfg, r.controllerName, false)
		r.restConfig = cfg
	}
	r.logger.WithField("cache-dir", r.cacheDir).Debug("creating cmdutil.Factory from REST client config and cache directory")
	f := cmdutil.NewFactory(&restConfigClientGetter{restConfig: r.restConfig, cacheDir: r.cacheDir, namespace: namespace})
	return f, nil
}

type restConfigClientGetter struct {
	restConfig *rest.Config
	cacheDir   string
	namespace  string
}

// ToRESTConfig returns restconfig
func (r *restConfigClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.restConfig, nil
}

// ToDiscoveryClient returns discovery client
func (r *restConfigClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config := rest.CopyConfig(r.restConfig)
	return getDiscoveryClient(config, r.cacheDir)
}

// ToRESTMapper returns a restmapper
func (r *restConfigClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient)
	return expander, nil
}

// ToRawKubeConfigLoader return kubeconfig loader as-is
func (r *restConfigClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	cfg := GenerateClientConfigFromRESTConfig("default", r.restConfig)
	overrides := &clientcmd.ConfigOverrides{}
	if len(r.namespace) > 0 {
		overrides.Context.Namespace = r.namespace
	}
	return clientcmd.NewNonInteractiveClientConfig(*cfg, "", overrides, nil)
}

// https://github.com/openshift/hive/blob/master/pkg/resource/serializer.go
var (
	jsonAPI json.API
)

func init() {
	jsonAPI = json.Config{EscapeHTML: true}.Froze()
	jsonAPI.RegisterExtension(&metaTimeExtension{})
}

type metaTimeExtension struct {
	json.DummyExtension
}

func (e *metaTimeExtension) CreateEncoder(typ reflect2.Type) json.ValEncoder {
	if typ.Type1() == reflect.TypeOf(metav1.Time{}) {
		return e
	}
	return nil
}

func (e *metaTimeExtension) IsEmpty(ptr unsafe.Pointer) bool {
	metaTime := reflect2.TypeOf(metav1.Time{}).UnsafeIndirect(ptr).(metav1.Time)
	return metaTime.IsZero()
}

func (e *metaTimeExtension) Encode(ptr unsafe.Pointer, stream *json.Stream) {
	metaTime := reflect2.TypeOf(metav1.Time{}).UnsafeIndirect(ptr).(metav1.Time)
	data, err := metaTime.MarshalJSON()
	if err != nil {
		log.Warnf("cannot marshal %#v as meta time: %v", ptr, err)
		return
	}
	_, err = stream.Write(data)
	if err != nil {
		log.Warnf("cannot write serialized time (%s): %v", string(data), err)
	}
}

type jsonPrinter struct{}

func (p *jsonPrinter) PrintObj(obj runtime.Object, w io.Writer) error {
	data, err := jsonAPI.Marshal(obj)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

// Serialize uses a custom JSON extension to properly determine whether metav1.Time should
// be serialized or not. In cases where a metav1.Time is labeled as 'omitempty', the default
// json marshaler still outputs a "null" value because it is considered a struct.
// The json-iterator/go marshaler will first check whether a value is empty and if its tag
// says 'omitempty' it will not output it.
// This is needed for us to prevent patching from happening unnecessarily when applying resources
// that don't have a timeCreated timestamp. With the default serializer, they output a
// `timeCreated: null` which always causes a mismatch with whatever's already in the server.
func Serialize(obj runtime.Object, scheme *runtime.Scheme) ([]byte, error) {
	printer := printers.NewTypeSetter(scheme).ToPrinter(&jsonPrinter{})
	buf := &bytes.Buffer{}
	if err := printer.PrintObj(obj, buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
