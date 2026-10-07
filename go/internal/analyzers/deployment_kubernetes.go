package analyzers

import "strings"

//	{
//	  責務: [
//	    ParseKubernetes: Kubernetes workload・Service・Ingressをdeployment factsへ変換します
//	  ]
//	  処理: [
//	    1: YAML documentからconfirmed nodeを作る
//	    2: selectorとbackendからroutes_to relationを解決する
//	    3: 安定順に返す
//	  ]
//	  引数: [
//	    source: Kubernetes manifest YAML本文
//	    sourceName: manifestのproject相対path
//	  ]
//	  戻り値: [
//	    DeploymentTopology: 抽出したfacts
//	    error: YAMLが不正な場合の解析error
//	  ]
//	}
func ParseKubernetes(source, sourceName string) (DeploymentTopology, error) {
	documents, err := decodeYAMLDocuments(source, sourceName)
	if err != nil {
		return DeploymentTopology{}, err
	}
	topology := emptyDeploymentTopology()
	workloadLabels := make(map[string]map[string]any)
	serviceNames := make(map[string]struct{})
	// Workload labelとservice名を収集し、次段のrelation解決に備えます。
	for _, document := range documents {
		kind, name := yamlValueString(document["kind"]), yamlMetadataName(document)
		if name == "" {
			continue
		}
		spec, _ := document["spec"].(map[string]any)
		switch kind {
		case "Deployment", "StatefulSet", "DaemonSet", "Pod", "Job", "CronJob":
			nodeID, metadata := kubernetesWorkload(document, spec, kind, name)
			workloadLabels[nodeID] = kubernetesWorkloadLabels(document, spec)
			topology.Nodes = append(topology.Nodes, DeploymentNode{
				ID: nodeID, Label: name, Kind: "workload", Confidence: ConfidenceConfirmed,
				Environment: "kubernetes", Source: sourceName, Metadata: metadata,
			})
		case "Service":
			serviceNames[name] = struct{}{}
			topology.Nodes = append(topology.Nodes, DeploymentNode{
				ID: "k8s:service:" + name, Label: name, Kind: "service-endpoint",
				Confidence: ConfidenceConfirmed, Environment: "kubernetes", Source: sourceName,
			})
		case "Ingress":
			topology.Nodes = append(topology.Nodes, DeploymentNode{
				ID: "k8s:ingress:" + name, Label: name, Kind: "ingress",
				Confidence: ConfidenceConfirmed, Environment: "kubernetes", Source: sourceName,
			})
		}
	}
	// 全nodeを収集した後にselectorとIngress backendを解決します。
	for _, document := range documents {
		kind, name := yamlValueString(document["kind"]), yamlMetadataName(document)
		spec, _ := document["spec"].(map[string]any)
		switch kind {
		case "Service":
			selector, _ := spec["selector"].(map[string]any)
			for workloadID, labels := range workloadLabels {
				if yamlSelectorMatches(selector, labels) {
					topology.Connections = append(topology.Connections, DeploymentConnection{
						Source: "k8s:service:" + name, Target: workloadID,
						Relation: "routes_to", Confidence: ConfidenceConfirmed,
					})
				}
			}
		case "Ingress":
			for _, service := range kubernetesIngressServices(spec) {
				if _, exists := serviceNames[service]; exists {
					topology.Connections = append(topology.Connections, DeploymentConnection{
						Source: "k8s:ingress:" + name, Target: "k8s:service:" + service,
						Relation: "routes_to", Confidence: ConfidenceConfirmed,
					})
				}
			}
		}
	}
	return sortDeploymentTopology(topology), nil
}

//	{
//	  責務: [
//	    yamlMetadataName: documentからmetadata.nameを取得します
//	  ]
//	  処理: [
//	    1: metadata mappingを読む
//	    2: nameをstringにする
//	  ]
//	  引数: [
//	    document: Kubernetes YAML document
//	  ]
//	  戻り値: [
//	    string: metadata.nameまたは空文字
//	  ]
//	}
func yamlMetadataName(document map[string]any) string {
	metadata, _ := document["metadata"].(map[string]any)
	value, exists := metadata["name"]
	if !exists {
		return ""
	}
	return yamlValueString(value)
}

//	{
//	  責務: [
//	    kubernetesWorkload: workloadのimage・kind・replica metadataを抽出します
//	  ]
//	  処理: [
//	    1: pod templateを解決する
//	    2: container imageを集める
//	    3: logical node IDとmetadataを作る
//	  ]
//	  引数: [
//	    document: Kubernetes document
//	    spec: workload spec
//	    kind: workload kind
//	    name: workload name
//	  ]
//	  戻り値: [
//	    nodeID: 確定したlogical ID
//	    metadata: workloadの表示metadata
//	  ]
//	}
func kubernetesWorkload(document, spec map[string]any, kind, name string) (string, map[string]string) {
	template, _ := spec["template"].(map[string]any)
	if template == nil {
		template = document
	}
	podSpec, _ := template["spec"].(map[string]any)
	if podSpec == nil {
		podSpec = spec
	}
	images := make([]string, 0)
	containers, _ := podSpec["containers"].([]any)
	for _, rawContainer := range containers {
		container, _ := rawContainer.(map[string]any)
		if image := yamlValueString(container["image"]); image != "" {
			images = append(images, image)
		}
	}
	metadata := map[string]string{"kind": kind}
	if len(images) > 0 {
		metadata["images"] = strings.Join(images, ", ")
	}
	if replicas := yamlValueString(spec["replicas"]); replicas != "" {
		metadata["replicas"] = replicas
	}
	return "k8s:" + strings.ToLower(kind) + ":" + name, metadata
}

//	{
//	  責務: [
//	    kubernetesWorkloadLabels: workloadのpod labelを取得します
//	  ]
//	  処理: [
//	    1: pod templateを解決する
//	    2: metadata.labels mappingを返す
//	  ]
//	  引数: [
//	    document: Kubernetes document
//	    spec: workload spec
//	  ]
//	  戻り値: [
//	    mapping: pod labelまたは空mapping
//	  ]
//	}
func kubernetesWorkloadLabels(document, spec map[string]any) map[string]any {
	template, _ := spec["template"].(map[string]any)
	if template == nil {
		template = document
	}
	metadata, _ := template["metadata"].(map[string]any)
	labels, _ := metadata["labels"].(map[string]any)
	return labels
}

//	{
//	  責務: [
//	    yamlSelectorMatches: Service selectorがすべてworkload labelに一致するか判定します
//	  ]
//	  処理: [
//	    1: 空selectorを不一致とする
//	    2: keyごとの値をstring比較する
//	  ]
//	  引数: [
//	    selector: Service selector
//	    labels: workload label
//	  ]
//	  戻り値: [
//	    bool: 全selector keyが一致する場合true
//	  ]
//	}
func yamlSelectorMatches(selector, labels map[string]any) bool {
	if len(selector) == 0 {
		return false
	}
	for key, value := range selector {
		if yamlValueString(labels[key]) != yamlValueString(value) {
			return false
		}
	}
	return true
}

//	{
//	  責務: [
//	    kubernetesIngressServices: Ingress backendが参照するservice名を抽出します
//	  ]
//	  処理: [
//	    1: defaultBackendを読む
//	    2: HTTP rule path backendを読む
//	    3: 重複を除いたnameを返す
//	  ]
//	  引数: [
//	    spec: Ingress spec
//	  ]
//	  戻り値: [
//	    string列: 参照service名
//	  ]
//	}
func kubernetesIngressServices(spec map[string]any) []string {
	services := make(map[string]struct{})
	defaultBackend, _ := spec["defaultBackend"].(map[string]any)
	addService := func(backend map[string]any) {
		service, _ := backend["service"].(map[string]any)
		if name := yamlValueString(service["name"]); name != "" {
			services[name] = struct{}{}
		}
	}
	addService(defaultBackend)
	rules, _ := spec["rules"].([]any)
	for _, rawRule := range rules {
		rule, _ := rawRule.(map[string]any)
		http, _ := rule["http"].(map[string]any)
		paths, _ := http["paths"].([]any)
		for _, rawPath := range paths {
			path, _ := rawPath.(map[string]any)
			backend, _ := path["backend"].(map[string]any)
			addService(backend)
		}
	}
	result := make([]string, 0, len(services))
	for name := range services {
		result = append(result, name)
	}
	return result
}
