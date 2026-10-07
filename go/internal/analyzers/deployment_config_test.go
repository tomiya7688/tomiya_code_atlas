package analyzers

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

// deploymentConfigFixturePathは、Python版とGo版で比較するdeployment設定fixtureのpathです。
const deploymentConfigFixturePath = "../../../tests/fixtures/analyzers/deployment_config_v1.json"

//	{
//	  責務: [
//	    TestDeploymentConfigExtractorsMatchSharedGolden: 3種類のconfig extractorを共有goldenと比較します
//	  ]
//	  処理: [
//	    1: fixtureを読む
//	    2: Dockerfile・Compose・Kubernetesを解析する
//	    3: nodeとrelationをgoldenと照合する
//	  ]
//	  引数: [
//	    t: Go test runner
//	  ]
//	  戻り値: [
//	    なし: golden不一致や解析errorをtest failureにする
//	  ]
//	}
func TestDeploymentConfigExtractorsMatchSharedGolden(t *testing.T) {
	var fixture struct {
		Dockerfile struct {
			SourceName string             `json:"source_name"`
			Source     string             `json:"source"`
			Expected   DeploymentTopology `json:"expected"`
		} `json:"dockerfile"`
		Compose struct {
			SourceName string             `json:"source_name"`
			Source     string             `json:"source"`
			Expected   DeploymentTopology `json:"expected"`
		} `json:"compose"`
		Kubernetes struct {
			SourceName string             `json:"source_name"`
			Source     string             `json:"source"`
			Expected   DeploymentTopology `json:"expected"`
		} `json:"kubernetes"`
	}
	data, err := os.ReadFile(deploymentConfigFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	assertDeploymentTopology(t, "Dockerfile", ParseDockerfile(fixture.Dockerfile.Source, fixture.Dockerfile.SourceName), fixture.Dockerfile.Expected)

	compose, err := ParseCompose(fixture.Compose.Source, fixture.Compose.SourceName)
	if err != nil {
		t.Fatal(err)
	}
	assertDeploymentTopology(t, "Compose", compose, fixture.Compose.Expected)

	kubernetes, err := ParseKubernetes(fixture.Kubernetes.Source, fixture.Kubernetes.SourceName)
	if err != nil {
		t.Fatal(err)
	}
	assertDeploymentTopology(t, "Kubernetes", kubernetes, fixture.Kubernetes.Expected)
}

//	{
//	  責務: [
//	    TestDeploymentConfigExtractorsRejectInvalidYAML: 不正YAMLを成功扱いしないことを確認します
//	  ]
//	  処理: [
//	    1: ComposeとKubernetesへ不正文書を渡す
//	    2: errorとsource nameを確認する
//	  ]
//	  引数: [
//	    t: Go test runner
//	  ]
//	  戻り値: [
//	    なし: error契約が破れた場合をtest failureにする
//	  ]
//	}
func TestDeploymentConfigExtractorsRejectInvalidYAML(t *testing.T) {
	for name, parse := range map[string]func(string, string) (DeploymentTopology, error){
		"Compose":    ParseCompose,
		"Kubernetes": ParseKubernetes,
	} {
		_, err := parse("services: [", "broken.yaml")
		if err == nil || !strings.Contains(err.Error(), "broken.yaml") {
			t.Errorf("%s error = %v, want source name in parse error", name, err)
		}
	}
}

//	{
//	  責務: [
//	    TestEmptyDeploymentConfigUsesJSONArrays: 空入力と空documentで空のJSON arrayを返すことを確認します
//	  ]
//	  処理: [
//	    1: Dockerfile・Compose・Kubernetesの空入力を解析する
//	    2: nodeとconnectionが空sliceであることを確認する
//	    3: nameのないresourceを無視する
//	  ]
//	  引数: [
//	    t: Go test runner
//	  ]
//	  戻り値: [
//	    なし: empty output contractが破れた場合をtest failureにする
//	  ]
//	}
func TestEmptyDeploymentConfigUsesJSONArrays(t *testing.T) {
	expected := DeploymentTopology{Nodes: []DeploymentNode{}, Connections: []DeploymentConnection{}}
	assertDeploymentTopology(t, "empty Dockerfile", ParseDockerfile("", "Dockerfile"), expected)
	assertDeploymentTopology(t, "empty merge", MergeDeploymentTopologies(), expected)
	assertDeploymentTopology(t, "empty component graph", DeploymentTopologyFromComponentGraph(BuildComponentDependencyGraph(nil)), expected)

	for name, parse := range map[string]func(string, string) (DeploymentTopology, error){
		"Compose":    ParseCompose,
		"Kubernetes": ParseKubernetes,
	} {
		got, err := parse("---\nnull\n---\n", "empty.yaml")
		if err != nil {
			t.Errorf("%s parse error = %v", name, err)
			continue
		}
		assertDeploymentTopology(t, "empty "+name, got, expected)
	}

	kubernetes, err := ParseKubernetes("kind: Service\nmetadata: {}\n", "missing-name.yaml")
	if err != nil {
		t.Fatal(err)
	}
	assertDeploymentTopology(t, "missing Kubernetes name", kubernetes, expected)
}

//	{
//	  責務: [
//	    assertDeploymentTopology: deployment topologyの一致を検証します
//	  ]
//	  処理: [
//	    1: gotとwantを比較する
//	    2: 不一致の場合に両方を報告する
//	  ]
//	  引数: [
//	    t: Go test runner
//	    name: 比較対象名
//	    got: 実際のtopology
//	    want: golden topology
//	  ]
//	  戻り値: [
//	    なし: 比較結果をtest runnerへ報告する
//	  ]
//	}
func assertDeploymentTopology(t *testing.T, name string, got, want DeploymentTopology) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s topology = %#v, want %#v", name, got, want)
	}
}
