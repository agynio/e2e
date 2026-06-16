//go:build e2e && svc_egress_gateway

package tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	egressGatewayWiringTestTimeout   = 3 * time.Minute
	egressManagedByValue             = "agents-orchestrator"
	egressGatewayHealthAddr          = "egress-gateway:50051"
	egressGatewayHealthTimeout       = 30 * time.Second
	egressNetworkPolicyName          = "agent-workload-egress"
	egressCASecretName               = "egress-ca"
	egressGatewayIdentitySecret      = "egress-gateway-ziti-identity"
	egressZitiEnrollmentSecret       = "egress-gateway-enrollment"
	egressCACertPath                 = "/etc/agyn/egress-ca/ca.crt"
	egressOpenZitiCIDR               = "100.64.0.0/10"
	egressPublicInternetCIDR         = "0.0.0.0/0"
	egressExpectedClusterPodCIDR     = "EGRESS_EXPECTED_CLUSTER_POD_CIDR"
	egressExpectedClusterServiceCIDR = "EGRESS_EXPECTED_CLUSTER_SERVICE_CIDR"
	egressExpectedAdditionalCIDRs    = "EGRESS_EXPECTED_ADDITIONAL_INTERNAL_CIDRS"
)

var egressDefaultBlockedCIDRs = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"127.0.0.0/8",
}

func TestEgressGatewayDeploymentWiring(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), egressGatewayWiringTestTimeout)
	t.Cleanup(cancel)

	waitForEgressGatewayHealth(t, ctx)
	assertEgressGatewaySecrets(t, ctx)
	assertEgressWorkloadNetworkPolicy(t, ctx)
	assertEgressCAInlineWorkloadPath(t, ctx)
}

func waitForEgressGatewayHealth(t *testing.T, ctx context.Context) {
	t.Helper()
	endpoint := "http://" + envOrDefault("EGRESS_GATEWAY_HEALTH_ADDRESS", egressGatewayHealthAddr) + "/healthz"
	client := &http.Client{Timeout: 5 * time.Second}
	pollCtx, pollCancel := context.WithTimeout(ctx, egressGatewayHealthTimeout)
	defer pollCancel()
	if err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected health status %d", response.StatusCode)
		}
		return nil
	}); err != nil {
		t.Fatalf("wait for egress gateway health: %v", err)
	}
}

func assertEgressGatewaySecrets(t *testing.T, ctx context.Context) {
	t.Helper()
	clientset := egressKubeClientset(t)
	namespace := platformNamespace()
	caSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, egressCASecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get %s/%s: %v", namespace, egressCASecretName, err)
	}
	if len(caSecret.Data[corev1.TLSCertKey]) == 0 || len(caSecret.Data[corev1.TLSPrivateKeyKey]) == 0 {
		t.Fatalf("%s/%s must include tls.crt and tls.key", namespace, egressCASecretName)
	}
	identitySecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, egressGatewayIdentitySecret, metav1.GetOptions{})
	if err == nil {
		if len(identitySecret.Data["identity.json"]) == 0 {
			t.Fatalf("%s/%s must include identity.json", namespace, egressGatewayIdentitySecret)
		}
		return
	}
	enrollmentSecret, enrollmentErr := clientset.CoreV1().Secrets(namespace).Get(ctx, egressZitiEnrollmentSecret, metav1.GetOptions{})
	if enrollmentErr != nil {
		t.Fatalf("get egress gateway identity secret %s/%s: %v; enrollment secret %s/%s: %v", namespace, egressGatewayIdentitySecret, err, namespace, egressZitiEnrollmentSecret, enrollmentErr)
	}
	if len(enrollmentSecret.Data["enrollmentJwt"]) == 0 {
		t.Fatalf("%s/%s must include enrollmentJwt", namespace, egressZitiEnrollmentSecret)
	}
}

func assertEgressWorkloadNetworkPolicy(t *testing.T, ctx context.Context) {
	t.Helper()
	policy, err := egressKubeClientset(t).NetworkingV1().NetworkPolicies(egressWorkloadNamespace(t)).Get(ctx, egressNetworkPolicyName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get workload egress network policy: %v", err)
	}
	if policy.Spec.PolicyTypes == nil || !networkPolicyHasType(policy.Spec.PolicyTypes, networkingv1.PolicyTypeEgress) {
		t.Fatalf("network policy %s must include Egress policy type", egressNetworkPolicyName)
	}
	if policy.Spec.PodSelector.MatchLabels["agyn.dev/managed-by"] != egressManagedByValue {
		t.Fatalf("network policy selector mismatch: %+v", policy.Spec.PodSelector.MatchLabels)
	}
	if !networkPolicyAllowsCIDR(policy, egressOpenZitiCIDR) {
		t.Fatalf("network policy %s must allow OpenZiti CIDR %s", egressNetworkPolicyName, egressOpenZitiCIDR)
	}
	if !networkPolicyAllowsDNS(policy) {
		t.Fatalf("network policy %s must allow DNS", egressNetworkPolicyName)
	}
	expectedCIDRs := expectedEgressNetworkPolicyExcludedCIDRs()
	assertNetworkPolicyAllowsPublicInternetExceptCIDRs(t, policy, expectedCIDRs)
}

func assertEgressCAInlineWorkloadPath(t *testing.T, ctx context.Context) {
	t.Helper()
	caSecret, err := egressKubeClientset(t).CoreV1().Secrets(platformNamespace()).Get(ctx, egressCASecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get egress ca secret: %v", err)
	}
	caBytes := caSecret.Data[corev1.TLSCertKey]
	if len(caBytes) == 0 {
		t.Fatalf("%s/%s missing %s", platformNamespace(), egressCASecretName, corev1.TLSCertKey)
	}
	runnerClient := newK8sRunnerClient(t)
	request := sleepWorkloadRequest()
	request.AdditionalProperties = map[string]string{"label.agyn.dev/managed-by": egressManagedByValue}
	request.InlineFiles = map[string][]byte{egressCACertPath: caBytes}
	request.Main.InlineFileMounts = []*runnerv1.InlineFileMount{{Path: egressCACertPath}}
	request.Main.Env = []*runnerv1.EnvVar{
		{Name: "SSL_CERT_FILE", Value: egressCACertPath},
		{Name: "REQUESTS_CA_BUNDLE", Value: egressCACertPath},
		{Name: "NODE_EXTRA_CA_CERTS", Value: egressCACertPath},
		{Name: "CURL_CA_BUNDLE", Value: egressCACertPath},
		{Name: "SSL_CERT_DIR", Value: "/etc/agyn/egress-ca"},
	}
	resp := startWorkloadWithCleanup(t, ctx, runnerClient, request)
	waitRunning(t, ctx, runnerClient, resp.GetId())
	result := collectExecOutput(t, ctx, runnerClient, &runnerv1.ExecStartRequest{
		TargetId:     resp.GetContainers().GetMain(),
		CommandShell: fmt.Sprintf("test -s %[1]s && test \"$SSL_CERT_FILE\" = %[1]s && test \"$CURL_CA_BUNDLE\" = %[1]s", egressCACertPath),
	})
	if result.exit == nil || result.exit.GetExitCode() != 0 {
		t.Fatalf("egress CA workload contract check failed: exit=%v stdout=%q stderr=%q", result.exit, result.stdout, result.stderr)
	}
}

func expectedEgressNetworkPolicyExcludedCIDRs() []string {
	cidrs := append([]string{}, egressDefaultBlockedCIDRs...)
	for _, key := range []string{egressExpectedClusterPodCIDR, egressExpectedClusterServiceCIDR, egressExpectedAdditionalCIDRs} {
		for _, cidr := range strings.Split(os.Getenv(key), ",") {
			trimmed := strings.TrimSpace(cidr)
			if trimmed != "" {
				cidrs = append(cidrs, trimmed)
			}
		}
	}
	return cidrs
}

func networkPolicyHasType(types []networkingv1.PolicyType, expected networkingv1.PolicyType) bool {
	for _, candidate := range types {
		if candidate == expected {
			return true
		}
	}
	return false
}

func networkPolicyAllowsCIDR(policy *networkingv1.NetworkPolicy, cidr string) bool {
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.IPBlock != nil && peer.IPBlock.CIDR == cidr {
				return true
			}
		}
	}
	return false
}

func assertNetworkPolicyAllowsPublicInternetExceptCIDRs(t *testing.T, policy *networkingv1.NetworkPolicy, blockedCIDRs []string) {
	t.Helper()
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.IPBlock == nil || peer.IPBlock.CIDR != egressPublicInternetCIDR {
				continue
			}
			missing := missingCIDRs(peer.IPBlock.Except, blockedCIDRs)
			if len(missing) == 0 {
				return
			}
			t.Fatalf("network policy %s public internet %s missing configured CIDR exceptions %v; actual exceptions %v", egressNetworkPolicyName, egressPublicInternetCIDR, missing, peer.IPBlock.Except)
		}
	}
	t.Fatalf("network policy %s must allow public internet %s with configured CIDR exceptions %v", egressNetworkPolicyName, egressPublicInternetCIDR, blockedCIDRs)
}

func missingCIDRs(actual []string, expected []string) []string {
	actualSet := make(map[string]struct{}, len(actual))
	for _, cidr := range actual {
		actualSet[cidr] = struct{}{}
	}
	missing := make([]string, 0)
	for _, cidr := range expected {
		if _, ok := actualSet[cidr]; !ok {
			missing = append(missing, cidr)
		}
	}
	return missing
}

func networkPolicyAllowsDNS(policy *networkingv1.NetworkPolicy) bool {
	for _, rule := range policy.Spec.Egress {
		udp53 := false
		tcp53 := false
		for _, port := range rule.Ports {
			if port.Port == nil || port.Port.Type != intstr.Int || port.Port.IntVal != 53 || port.Protocol == nil {
				continue
			}
			switch *port.Protocol {
			case corev1.ProtocolUDP:
				udp53 = true
			case corev1.ProtocolTCP:
				tcp53 = true
			}
		}
		if udp53 && tcp53 {
			return true
		}
	}
	return false
}

func platformNamespace() string {
	return envOrDefault("E2E_NAMESPACE", envOrDefault("DEVSPACE_NAMESPACE", "platform"))
}

func egressKubeClientset(t *testing.T) *kubernetes.Clientset {
	t.Helper()
	config, err := rest.InClusterConfig()
	if err != nil {
		t.Fatalf("load in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("create kubernetes clientset: %v", err)
	}
	return clientset
}

func egressWorkloadNamespace(t *testing.T) string {
	t.Helper()
	return envOrDefault("WORKLOAD_NAMESPACE", "agyn-workloads")
}
