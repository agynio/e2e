//go:build e2e && svc_egress && svc_egress_gateway

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	egressv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/egress/v1"
	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	zitimgmtv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/ziti_management/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	egressDataPlaneTestTimeout       = 5 * time.Minute
	egressPostmanEchoHost            = "postman-echo.com"
	egressPostmanEchoPort            = int32(443)
	egressCurlImage                  = "curlimages/curl:8.17.0"
	egressZitiSidecarImageEnvKey     = "ZITI_SIDECAR_IMAGE"
	egressDefaultZitiSidecarImage    = "openziti/ziti-tunnel:2.0.0-pre8"
	egressWorkloadDNSUpstreamEnvKey  = "WORKLOAD_DNS_UPSTREAM"
	egressDefaultWorkloadDNSUpstream = "10.96.0.10"
	egressZitiControllerHostEnvKey   = "ZITI_CONTROLLER_HOST"
	egressDefaultZitiControllerHost  = "ziti.agyn.dev"
	egressZitiIdentityVolumeName     = "ziti-identity"
	egressZitiIdentityMountPath      = "/netfoundry"
	egressZitiIdentityBasename       = "agent"
	egressZitiEnrollContainerName    = "ziti-enroll"
	egressZitiSidecarContainerName   = "ziti-sidecar"
	egressZitiRequiredNetAdmin       = "NET_ADMIN"
	egressZitiEnrollmentTokenEnvVar  = "ZITI_ENROLL_TOKEN"
	egressZitiIdentityBasenameEnvVar = "ZITI_IDENTITY_BASENAME"
	egressZitiIdentityDirEnvVar      = "ZITI_IDENTITY_DIR"
)

func TestEgressGatewayDataPlaneSecretInjection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), egressDataPlaneTestTimeout)
	t.Cleanup(cancel)

	fixture := setupEgressFixture(t, ctx)
	tokenValue := "e2e-egress-token-" + uuid.NewString()
	queryMarker := uuid.NewString()

	secret := createEgressSecretWithValue(t, fixture.userCtx, fixture.secrets, fixture.organizationID, tokenValue)
	secretID := secret.GetMeta().GetId()
	t.Cleanup(func() { deleteEgressSecret(t, fixture.userCtx, fixture.secrets, secretID) })

	allowRule := createPostmanEchoEgressRule(t, fixture.userCtx, fixture, secretID)
	allowRuleID := allowRule.GetMeta().GetId()
	t.Cleanup(func() { deleteEgressRule(t, fixture.userCtx, fixture.egress, allowRuleID) })

	attachment := createEgressRuleAttachment(t, fixture.userCtx, fixture.egress, allowRuleID, fixture.agentID)
	attachmentID := attachment.GetMeta().GetId()
	t.Cleanup(func() { deleteEgressRuleAttachment(t, fixture.userCtx, fixture.egress, attachmentID) })

	waitForEgressRuleByAgent(t, ctx, fixture.egress, fixture.agentID, allowRuleID)

	runnerClient := newK8sRunnerClient(t)
	zitiIdentityID, enrollmentJWT := createEgressZitiAgentIdentity(t, ctx, fixture.agentID)
	t.Cleanup(func() { deleteEgressZitiIdentity(t, zitiIdentityID) })
	logEgressEnrollmentDiagnostics(t, ctx, zitiIdentityID, enrollmentJWT)

	request := postmanEchoWorkloadRequest(t, ctx, enrollmentJWT, queryMarker)
	response := startWorkloadWithCleanup(t, ctx, runnerClient, request)
	workloadID := response.GetId()
	waitRunning(t, ctx, runnerClient, workloadID)
	targetID := response.GetContainers().GetMain()
	if targetID == "" {
		t.Fatal("start postman echo workload: missing main container target id")
	}

	execResult := waitForPostmanEchoOutput(t, ctx, runnerClient, targetID)
	if execResult.exit == nil || execResult.exit.GetExitCode() != 0 {
		t.Fatalf("postman echo request failed: exit=%v stdout=%q stderr=%q", execResult.exit, execResult.stdout, execResult.stderr)
	}
	assertPostmanEchoAuthorization(t, execResult.stdout, queryMarker, tokenValue)
}

func createPostmanEchoEgressRule(t *testing.T, ctx context.Context, fixture egressFixture, secretID string) *egressv1.EgressRule {
	t.Helper()
	resp, err := fixture.egress.CreateEgressRule(ctx, &egressv1.CreateEgressRuleRequest{
		OrganizationId: fixture.organizationID,
		Name:           "e2e-egress-postman-echo-" + uuid.NewString(),
		Description:    "E2E Egress Gateway data-plane Postman Echo rule",
		Matcher: &egressv1.EgressRuleMatcher{
			DomainPattern: egressPostmanEchoHost,
			Ports:         []int32{egressPostmanEchoPort},
			Methods:       []string{"GET"},
			PathPattern:   "/get",
		},
		Effect: &egressv1.EgressRuleEffect{
			Action: egressv1.EgressRuleAction_EGRESS_RULE_ACTION_ALLOW.Enum(),
			Inject: []*egressv1.EgressRuleHeader{{
				Name:       "Authorization",
				Scheme:     egressv1.HeaderAuthScheme_HEADER_AUTH_SCHEME_BEARER,
				Credential: &egressv1.EgressRuleHeader_SecretId{SecretId: secretID},
			}},
		},
	})
	if err != nil {
		t.Fatalf("create postman echo egress rule: %v", err)
	}
	rule := resp.GetEgressRule()
	if rule == nil || rule.GetMeta() == nil || rule.GetMeta().GetId() == "" {
		t.Fatal("create postman echo egress rule: missing id")
	}
	return rule
}

func createEgressZitiAgentIdentity(t *testing.T, ctx context.Context, agentID string) (string, string) {
	t.Helper()
	conn := dialGRPC(t, zitiManagementAddr(t))
	client := zitimgmtv1.NewZitiManagementServiceClient(conn)
	resp, err := client.CreateAgentIdentity(ctx, &zitimgmtv1.CreateAgentIdentityRequest{
		AgentId:    agentID,
		WorkloadId: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("create egress ziti agent identity: %v", err)
	}
	zitiIdentityID := strings.TrimSpace(resp.GetZitiIdentityId())
	enrollmentJWT := strings.TrimSpace(resp.GetEnrollmentJwt())
	if zitiIdentityID == "" || enrollmentJWT == "" {
		t.Fatalf("create egress ziti agent identity: missing id or enrollment jwt: id=%q jwt_present=%t", zitiIdentityID, enrollmentJWT != "")
	}
	return zitiIdentityID, enrollmentJWT
}

func deleteEgressZitiIdentity(t *testing.T, zitiIdentityID string) {
	t.Helper()
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn := dialGRPC(t, zitiManagementAddr(t))
	client := zitimgmtv1.NewZitiManagementServiceClient(conn)
	_, err := client.DeleteIdentity(cleanupCtx, &zitimgmtv1.DeleteIdentityRequest{ZitiIdentityId: zitiIdentityID})
	if err != nil && status.Code(err) != codes.NotFound {
		t.Logf("cleanup: delete ziti identity %s: %v", zitiIdentityID, err)
	}
}

func postmanEchoWorkloadRequest(t *testing.T, ctx context.Context, enrollmentJWT, queryMarker string) *runnerv1.StartWorkloadRequest {
	t.Helper()
	caBytes := egressCABytes(t, ctx)
	return &runnerv1.StartWorkloadRequest{
		Main: &runnerv1.ContainerSpec{
			Image:      egressCurlImage,
			Entrypoint: "/bin/sh",
			Cmd:        []string{"-c", postmanEchoCurlScript(queryMarker)},
			Env: []*runnerv1.EnvVar{
				{Name: "SSL_CERT_FILE", Value: egressCACertPath},
				{Name: "CURL_CA_BUNDLE", Value: egressCACertPath},
			},
			InlineFileMounts: []*runnerv1.InlineFileMount{{Path: egressCACertPath}},
		},
		Volumes: []*runnerv1.VolumeSpec{{
			Name: egressZitiIdentityVolumeName,
			Kind: runnerv1.VolumeKind_VOLUME_KIND_EPHEMERAL,
		}},
		InitContainers: []*runnerv1.ContainerSpec{
			egressZitiEnrollContainer(enrollmentJWT),
		},
		Sidecars: []*runnerv1.ContainerSpec{
			egressZitiSidecarContainer(),
		},
		InlineFiles: map[string][]byte{egressCACertPath: caBytes},
		AdditionalProperties: map[string]string{
			"label.agyn.dev/managed-by": egressManagedByValue,
		},
	}
}

func egressCABytes(t *testing.T, ctx context.Context) []byte {
	t.Helper()
	secret, err := egressKubeClientset(t).CoreV1().Secrets(platformNamespace()).Get(ctx, egressCASecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get egress ca secret: %v", err)
	}
	caBytes := secret.Data[corev1.TLSCertKey]
	if len(caBytes) == 0 {
		t.Fatalf("%s/%s missing %s", platformNamespace(), egressCASecretName, corev1.TLSCertKey)
	}
	return caBytes
}

func egressZitiEnrollContainer(enrollmentJWT string) *runnerv1.ContainerSpec {
	return &runnerv1.ContainerSpec{
		Image:      envOrDefault(egressZitiSidecarImageEnvKey, egressDefaultZitiSidecarImage),
		Name:       egressZitiEnrollContainerName,
		Entrypoint: "/usr/bin/bash",
		Cmd: []string{
			"-ec",
			egressZitiEnrollScript()},
		Env: []*runnerv1.EnvVar{
			{Name: egressZitiEnrollmentTokenEnvVar, Value: enrollmentJWT},
			{Name: egressZitiIdentityBasenameEnvVar, Value: egressZitiIdentityBasename},
			{Name: egressZitiIdentityDirEnvVar, Value: egressZitiIdentityMountPath},
			{Name: egressZitiControllerHostEnvKey, Value: envOrDefault(egressZitiControllerHostEnvKey, egressDefaultZitiControllerHost)},
		},
		Mounts: []*runnerv1.VolumeMount{{
			Volume:    egressZitiIdentityVolumeName,
			MountPath: egressZitiIdentityMountPath,
		}},
	}
}

func egressZitiSidecarContainer() *runnerv1.ContainerSpec {
	return &runnerv1.ContainerSpec{
		Image: envOrDefault(egressZitiSidecarImageEnvKey, egressDefaultZitiSidecarImage),
		Name:  egressZitiSidecarContainerName,
		Cmd: []string{
			"tproxy",
			"--dnsUpstream",
			"udp://" + envOrDefault(egressWorkloadDNSUpstreamEnvKey, egressDefaultWorkloadDNSUpstream) + ":53",
		},
		Env: []*runnerv1.EnvVar{
			{Name: egressZitiIdentityBasenameEnvVar, Value: egressZitiIdentityBasename},
			{Name: egressZitiIdentityDirEnvVar, Value: egressZitiIdentityMountPath},
		},
		Mounts: []*runnerv1.VolumeMount{{
			Volume:    egressZitiIdentityVolumeName,
			MountPath: egressZitiIdentityMountPath,
		}},
		RequiredCapabilities: []string{egressZitiRequiredNetAdmin},
	}
}

func egressZitiEnrollScript() string {
	return `identity_dir="${ZITI_IDENTITY_DIR}"
identity_basename="${ZITI_IDENTITY_BASENAME}"
identity_file="${identity_dir}/${identity_basename}.json"
jwt_file="${identity_dir}/${identity_basename}.jwt"
hosts_file="${ZITI_HOSTS_FILE:-/etc/hosts}"
ziti_controller_host="${ZITI_CONTROLLER_HOST}"
ziti_controller_ip="${ZITI_CONTROLLER_IP:-}"

mkdir -p "${identity_dir}"

if [[ ! -s "${identity_file}" ]]; then
  if [[ -z "${ZITI_ENROLL_TOKEN}" ]]; then
    echo "ZITI_ENROLL_TOKEN is required" >&2
    exit 1
  fi
  printf '%s\n' "${ZITI_ENROLL_TOKEN}" > "${jwt_file}"

  jwt_payload="${ZITI_ENROLL_TOKEN#*.}"
  jwt_payload="${jwt_payload%%.*}"
  jwt_payload="$(printf '%s' "${jwt_payload}" | tr '_-' '/+')"
  case $(( ${#jwt_payload} % 4 )) in
    2) jwt_payload="${jwt_payload}==" ;;
    3) jwt_payload="${jwt_payload}=" ;;
  esac
  jwt_payload_json="$(printf '%s' "${jwt_payload}" | base64 -d 2>/dev/null || true)"
  jwt_controller_host="$(printf '%s' "${jwt_payload_json}" | sed -nE 's/.*"iss"[[:space:]]*:[[:space:]]*"https?:\/\/([^"\/:]+).*/\1/p' | head -n 1)"
  if [[ -z "${ziti_controller_host}" ]]; then
    ziti_controller_host="${jwt_controller_host}"
  fi
  if [[ -n "${ziti_controller_host}" ]]; then
    awk -v host="${ziti_controller_host}" '($1 ~ /^(127\.|::1$)/) { for (i = 2; i <= NF; i++) if ($i == host) next } { print }' "${hosts_file}" > "${hosts_file}.tmp"
    cat "${hosts_file}.tmp" > "${hosts_file}"
    rm -f "${hosts_file}.tmp"
  fi
  if [[ -n "${ziti_controller_ip}" && -n "${ziti_controller_host}" ]]; then
    printf '%s %s\n' "${ziti_controller_ip}" "${ziti_controller_host}" >> "${hosts_file}"
  fi

  ziti edge enroll --jwt "${jwt_file}" --out "${identity_file}"
fi

if [[ ! -s "${identity_file}" ]]; then
  echo "expected identity file ${identity_file}" >&2
  exit 1
fi`
}

func postmanEchoCurlScript(queryMarker string) string {
	url := fmt.Sprintf("https://%s/get?egress_e2e=%s", egressPostmanEchoHost, queryMarker)
	return fmt.Sprintf("curl --fail --silent --show-error --location --connect-timeout 20 --max-time 60 %q > /tmp/postman-echo.json.tmp && mv /tmp/postman-echo.json.tmp /tmp/postman-echo.json; sleep 300", url)
}

func waitForPostmanEchoOutput(t *testing.T, ctx context.Context, client runnerv1.RunnerServiceClient, workloadID string) execResult {
	t.Helper()
	var result execResult
	pollCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		result = collectExecOutput(t, ctx, client, &runnerv1.ExecStartRequest{
			TargetId:     workloadID,
			CommandShell: "cat /tmp/postman-echo.json",
		})
		if result.exit == nil || result.exit.GetExitCode() != 0 {
			return fmt.Errorf("postman echo output unavailable: exit=%v stdout=%q stderr=%q", result.exit, result.stdout, result.stderr)
		}
		if strings.TrimSpace(result.stdout) == "" {
			return fmt.Errorf("postman echo output is empty")
		}
		if !json.Valid([]byte(result.stdout)) {
			return fmt.Errorf("postman echo output is not JSON: stdout=%q", result.stdout)
		}
		return nil
	}); err != nil {
		t.Fatalf("wait for postman echo output: %v", err)
	}
	return result
}

type postmanEchoResponse struct {
	Args    map[string]string `json:"args"`
	Headers map[string]string `json:"headers"`
}

func assertPostmanEchoAuthorization(t *testing.T, body, queryMarker, tokenValue string) {
	t.Helper()
	var echo postmanEchoResponse
	if err := json.Unmarshal([]byte(body), &echo); err != nil {
		t.Fatalf("parse postman echo response: %v body=%q", err, body)
	}
	if echo.Args["egress_e2e"] != queryMarker {
		t.Fatalf("postman echo query marker mismatch: got %q want %q body=%q", echo.Args["egress_e2e"], queryMarker, body)
	}
	wantAuthorization := "Bearer " + tokenValue
	if echo.Headers["authorization"] != wantAuthorization {
		t.Fatalf("postman echo authorization mismatch: got %q want %q", echo.Headers["authorization"], wantAuthorization)
	}
}
