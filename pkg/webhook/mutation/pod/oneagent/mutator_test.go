package oneagent

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testImage               = "test-image"
	testClusterID           = "test-cluster-id"
	testPodName             = "test-pod"
	testNamespaceName       = "test-namespace"
	testDynakubeName        = "test-dynakube"
	testLabelKeyMatching    = "inject"
	testLabelKeyNotMatching = "do-not-inject"
	testLabelValue          = "into-this-ns"
)

func TestEnabled(t *testing.T) {
	t.Run("turned off", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, map[string]string{dtwebhook.AnnotationOneAgentInject: "false"}, getTestNamespace(nil))

		enabled := mutator.Enabled(request.BaseRequest)

		require.False(t, enabled)
	})
	t.Run("on by default", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, getTestNamespace(nil))
		request.DynaKube.Spec.OneAgent.ApplicationMonitoring = &dynakube.ApplicationMonitoringSpec{}

		enabled := mutator.Enabled(request.BaseRequest)

		require.True(t, enabled)
	})
	t.Run("off by feature flag", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, getTestNamespace(nil))
		request.DynaKube.Annotations = map[string]string{dynakube.AnnotationFeatureAutomaticInjection: "false"}

		enabled := mutator.Enabled(request.BaseRequest)

		require.False(t, enabled)
	})
	t.Run("on with feature flag", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, getTestNamespace(nil))
		request.DynaKube.Spec.OneAgent.ApplicationMonitoring = &dynakube.ApplicationMonitoringSpec{}
		request.DynaKube.Annotations = map[string]string{dynakube.AnnotationFeatureAutomaticInjection: "true"}

		enabled := mutator.Enabled(request.BaseRequest)

		require.True(t, enabled)
	})
	t.Run("on with namespaceselector", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, getTestNamespaceWithMatchingLabel(nil, testLabelKeyMatching, testLabelValue))
		request.DynaKube.Annotations = map[string]string{dynakube.AnnotationFeatureAutomaticInjection: "true"}
		request.DynaKube = *addNamespaceSelector(&request.DynaKube)

		enabled := mutator.Enabled(request.BaseRequest)

		require.True(t, enabled)
	})
	t.Run("off due to not matching namespaceselector", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, getTestNamespaceWithMatchingLabel(nil, testLabelKeyNotMatching, testLabelValue))
		request.DynaKube.Annotations = map[string]string{dynakube.AnnotationFeatureAutomaticInjection: "true"}
		request.DynaKube = *addNamespaceSelector(&request.DynaKube)

		enabled := mutator.Enabled(request.BaseRequest)

		require.False(t, enabled)
	})
}

func TestInjected(t *testing.T) {
	t.Run("already marked", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, map[string]string{dtwebhook.AnnotationOneAgentInjected: "true"}, getTestNamespace(nil))

		enabled := mutator.Injected(request.BaseRequest)

		require.True(t, enabled)
	})
	t.Run("fresh", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, getTestNamespace(nil))

		enabled := mutator.Injected(request.BaseRequest)

		require.False(t, enabled)
	})
}

func TestEnsureInitSecret(t *testing.T) {
	t.Run("shouldn't create init secret if already there", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil, getTestNamespace(nil))

		err := mutator.ensureInitSecret(request)
		require.NoError(t, err)
	})
}

type mutateTestCase struct {
	name                                   string
	dk                                     dynakube.DynaKube
	expectedAdditionalEnvCount             int
	expectedAdditionalVolumeCount          int
	expectedAdditionalVolumeMountCount     int
	expectedAdditionalInitVolumeMountCount int
}

func TestMutate(t *testing.T) {
	testCases := []mutateTestCase{
		{
			name:                                   "basic, should mutate the pod and init container in the request",
			dk:                                     *getTestDynakube(),
			expectedAdditionalEnvCount:             2, // 1 deployment-metadata + 1 preload
			expectedAdditionalVolumeCount:          3, // bin, share, injection-config
			expectedAdditionalVolumeMountCount:     3, // 3 oneagent mounts(preload,bin,conf)
			expectedAdditionalInitVolumeMountCount: 3, // bin, share, injection-config
		},
		{
			name:                                   "everything turned on, should mutate the pod and init container in the request",
			dk:                                     *getTestComplexDynakube(),
			expectedAdditionalEnvCount:             5, // 1 deployment-metadata + 1 network-zone + 1 preload + 2 version-detection
			expectedAdditionalVolumeCount:          3, // bin, share, injection-config
			expectedAdditionalVolumeMountCount:     5, // 3 oneagent mounts(preload,bin,conf) + 1 cert mount + 1 curl-options
			expectedAdditionalInitVolumeMountCount: 3, // bin, share, injection-config
		},
		{
			name:                                   "basic + readonly-csi, should mutate the pod and init container in the request",
			dk:                                     *getTestReadOnlyCSIDynakube(),
			expectedAdditionalEnvCount:             2, // 1 deployment-metadata + 1 preload
			expectedAdditionalVolumeCount:          6, // bin, share, injection-config +  agent-conf, data-storage, agent-log
			expectedAdditionalVolumeMountCount:     6, // 3 oneagent mounts(preload,bin,conf) +3 oneagent mounts for readonly csi (agent-conf,data-storage,agent-log)
			expectedAdditionalInitVolumeMountCount: 4, // bin, share, injection-config, agent-conf
		},
	}

	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
			request := createTestMutationRequest(&testCases[index].dk, nil, getTestNamespace(nil))

			initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
			initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
			initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
			initialContainersLen := len(request.Pod.Spec.Containers)
			initialAnnotationsLen := len(request.Pod.Annotations)
			initialInitContainers := request.Pod.Spec.InitContainers

			err := mutator.Mutate(context.Background(), request)
			require.NoError(t, err)

			assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+testCase.expectedAdditionalEnvCount)
			assert.Len(t, request.Pod.Spec.Volumes, initialNumberOfVolumesLen+testCase.expectedAdditionalVolumeCount)
			assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+testCase.expectedAdditionalVolumeMountCount)

			assert.Len(t, initialInitContainers, len(request.Pod.Spec.InitContainers)) // the init container should be added when in the PodMutator
			assert.Equal(t, initialInitContainers, request.Pod.Spec.InitContainers)

			assert.Len(t, request.Pod.Annotations, initialAnnotationsLen+1) // +1 == injected-annotation

			assert.Len(t, request.InstallContainer.Env, 1+expectedBaseInitContainerEnvCount+(initialContainersLen*2))
			assert.Len(t, request.InstallContainer.VolumeMounts, testCase.expectedAdditionalInitVolumeMountCount)
		})
	}
}

func TestNoCommunicationHostsMutate(t *testing.T) {
	dk := getTestNoCommunicationHostDynakube()

	mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
	request := createTestMutationRequest(dk, nil, getTestNamespace(nil))

	initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
	initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
	initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
	initialAnnotationsLen := len(request.Pod.Annotations)
	initialInitContainers := request.Pod.Spec.InitContainers

	err := mutator.Mutate(context.Background(), request)
	require.NoError(t, err)

	assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen)
	assert.Len(t, request.Pod.Spec.Volumes, initialNumberOfVolumesLen)
	assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen)

	assert.Len(t, initialInitContainers, len(request.Pod.Spec.InitContainers)) // the init container should be added when in the PodMutator
	assert.Equal(t, initialInitContainers, request.Pod.Spec.InitContainers)

	assert.Len(t, request.Pod.Annotations, initialAnnotationsLen+2) // +2 == injected-annotation, reason-annotation
	require.Contains(t, request.Pod.Annotations, dtwebhook.AnnotationOneAgentInjected)
	require.Contains(t, request.Pod.Annotations, dtwebhook.AnnotationOneAgentReason)

	assert.Equal(t, "false", request.Pod.Annotations[dtwebhook.AnnotationOneAgentInjected])
	assert.Equal(t, dtwebhook.EmptyConnectionInfoReason, request.Pod.Annotations[dtwebhook.AnnotationOneAgentReason])

	assert.Empty(t, request.InstallContainer.Env)
	assert.Empty(t, request.InstallContainer.VolumeMounts)
}

type reinvokeTestCase struct {
	name                               string
	dk                                 dynakube.DynaKube
	expectedAdditionalEnvCount         int
	expectedAdditionalVolumeMountCount int
}

func TestReinvoke(t *testing.T) {
	testCases := []reinvokeTestCase{
		{
			name:                               "basic, should mutate the pod and init container in the request",
			dk:                                 *getTestDynakube(),
			expectedAdditionalEnvCount:         2, // 1 deployment-metadata + 1 preload
			expectedAdditionalVolumeMountCount: 3, // 3 oneagent mounts(preload,bin,conf)
		},
		{
			name:                               "everything turned on, should mutate the pod and init container in the request",
			dk:                                 *getTestComplexDynakube(),
			expectedAdditionalEnvCount:         5, // 1 deployment-metadata + 1 network-zone + 1 preload + 2 version-detection
			expectedAdditionalVolumeMountCount: 5, // 3 oneagent mounts(preload,bin,conf) + 1 cert mount + 1 curl-options
		},
		{
			name:                               "basic + readonly-csi, should mutate the pod and init container in the request",
			dk:                                 *getTestReadOnlyCSIDynakube(),
			expectedAdditionalEnvCount:         2, // 1 deployment-metadata + 1 preload
			expectedAdditionalVolumeMountCount: 6, // 3 oneagent mounts(preload,bin,conf) +3 oneagent mounts for readonly csi (agent-conf,data-storage,agent-log)
		},
	}

	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
			request := createTestReinvocationRequest(&testCases[index].dk, map[string]string{dtwebhook.AnnotationOneAgentInjected: "true"})

			initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
			initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
			initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
			initialContainersLen := len(request.Pod.Spec.Containers)
			initialAnnotationsLen := len(request.Pod.Annotations)

			updated := mutator.Reinvoke(request)
			require.True(t, updated)

			assert.Len(t, request.Pod.Spec.Volumes, initialNumberOfVolumesLen)
			assert.Len(t, request.Pod.Annotations, initialAnnotationsLen)

			assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+testCase.expectedAdditionalEnvCount)
			assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+testCase.expectedAdditionalVolumeMountCount)
			assert.Len(t, request.Pod.Spec.InitContainers[1].Env, 1+initialContainersLen*2) // +1 == installer mode
		})
	}

	t.Run("no change ==> no update", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := &dtwebhook.ReinvocationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				DynaKube: *getTestDynakube(),
				Pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{dtwebhook.AnnotationOneAgentInjected: "true"},
					},
				},
			},
		}
		updated := mutator.Reinvoke(request)
		require.False(t, updated)
	})
}

func createTestPodMutator(objects []client.Object) *Mutator {
	return &Mutator{
		client:           fake.NewClient(objects...),
		apiReader:        fake.NewClient(objects...),
		image:            testImage,
		clusterID:        testClusterID,
		webhookNamespace: testNamespaceName,
	}
}

func getTestInitSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.AgentInitSecretName,
			Namespace: testNamespaceName,
		},
	}
}

func addNamespaceSelector(dk *dynakube.DynaKube) *dynakube.DynaKube {
	dk.Spec.OneAgent.ApplicationMonitoring = &dynakube.ApplicationMonitoringSpec{}

	dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{
			testLabelKeyMatching: testLabelValue,
		},
	}

	return dk
}

func createTestMutationRequest(dk *dynakube.DynaKube, podAnnotations map[string]string, namespace corev1.Namespace) *dtwebhook.MutationRequest {
	if dk == nil {
		dk = &dynakube.DynaKube{}
	}

	return dtwebhook.NewMutationRequest(
		context.Background(),
		namespace,
		&corev1.Container{
			Name: dtwebhook.InstallContainerName,
		},
		getTestPod(podAnnotations),
		*dk,
	)
}

func createTestReinvocationRequest(dk *dynakube.DynaKube, annotations map[string]string) *dtwebhook.ReinvocationRequest {
	request := createTestMutationRequest(dk, annotations, getTestNamespace(nil)).ToReinvocationRequest()
	request.Pod.Spec.InitContainers = append(request.Pod.Spec.InitContainers, corev1.Container{Name: dtwebhook.InstallContainerName})

	return request
}

func getTestCSIDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: dynakube.OneAgentSpec{
				CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
			},
		},
		Status: getTestDynakubeCommunicationHostStatus(),
	}
}

func getTestReadOnlyCSIDynakube() *dynakube.DynaKube {
	dk := getTestCSIDynakube()
	dk.Annotations[dynakube.AnnotationFeatureReadOnlyCsiVolume] = "true"

	return dk
}

func getTestNoCommunicationHostDynakube() *dynakube.DynaKube {
	dk := getTestCSIDynakube()
	dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = []dynakube.CommunicationHostStatus{}

	return dk
}

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: dynakube.OneAgentSpec{
				ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{},
			},
		},
		Status: getTestDynakubeCommunicationHostStatus(),
	}
}

func getTestDynakubeWithContainerExclusion() *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: dynakube.OneAgentSpec{
				ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{},
			},
		},
		Status: getTestDynakubeCommunicationHostStatus(),
	}
	dk.ObjectMeta.Annotations[dtwebhook.AnnotationContainerInjection+"/sidecar-container"] = "false"

	return dk
}

func getTestDynakubeCommunicationHostStatus() dynakube.DynaKubeStatus {
	return dynakube.DynaKubeStatus{
		OneAgent: dynakube.OneAgentStatus{
			ConnectionInfoStatus: dynakube.OneAgentConnectionInfoStatus{
				CommunicationHosts: []dynakube.CommunicationHostStatus{
					{
						Protocol: "http",
						Host:     "dummyhost",
						Port:     666,
					},
				},
			},
		},
	}
}

func getTestDynakubeMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        testDynakubeName,
		Namespace:   testNamespaceName,
		Annotations: make(map[string]string),
	}
}

func getTestComplexDynakube() *dynakube.DynaKube {
	dk := getTestCSIDynakube()
	dk.Spec.Proxy = &dynakube.DynaKubeProxy{Value: "test-proxy"}
	dk.Spec.NetworkZone = "test-network-zone"
	dk.Spec.ActiveGate = dynakube.ActiveGateSpec{
		Capabilities:  []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName},
		TlsSecretName: "super-secret",
	}
	dk.Annotations = map[string]string{
		dynakube.AnnotationFeatureOneAgentInitialConnectRetry: "5",
		dynakube.AnnotationFeatureLabelVersionDetection:       "true",
	}

	return dk
}

func getTestPod(annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testPodName,
			Namespace:   testNamespaceName,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "main-container",
					Image: "alpine",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "volume",
							MountPath: "/volume",
						},
					},
				},
				{
					Name:  "sidecar-container",
					Image: "nginx",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "volume",
							MountPath: "/volume",
						},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:  "init-container",
					Image: "curlimages/curl",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

func getTestNamespace(annotations map[string]string) corev1.Namespace {
	return corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespaceName,
			Labels: map[string]string{
				dtwebhook.InjectionInstanceLabel: testDynakubeName,
			},
			Annotations: annotations,
		},
	}
}

func getTestNamespaceWithMatchingLabel(annotations map[string]string, labelKey, labelValue string) corev1.Namespace {
	return corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespaceName,
			Labels: map[string]string{
				dtwebhook.InjectionInstanceLabel: testDynakubeName,
				labelKey:                         labelValue,
			},
			Annotations: annotations,
		},
	}
}
