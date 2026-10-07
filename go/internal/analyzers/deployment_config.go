package analyzers

import "strings"

//	{
//	  責務: [
//	    ParseDockerfile: DockerfileのFROM宣言を確定したimage factsへ変換します
//	  ]
//	  処理: [
//	    1: FROM命令からbase imageを抽出する
//	    2: runtime・image nodeとFROM relationを構築する
//	    3: 安定順に並べて返す
//	  ]
//	  引数: [
//	    source: Dockerfile本文
//	    sourceName: deployment設定のproject相対path
//	  ]
//	  戻り値: [
//	    DeploymentTopology: 抽出した確定deployment facts
//	  ]
//	}
func ParseDockerfile(source, sourceName string) DeploymentTopology {
	bases := make([]string, 0)
	// FROM命令だけを集め、build stageの順序とmetadataを保持します。
	for _, rawLine := range strings.Split(source, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 || !strings.EqualFold(parts[0], "FROM") {
			continue
		}
		index := 1
		for index < len(parts) && strings.HasPrefix(parts[index], "--") {
			index++
		}
		if index < len(parts) {
			bases = append(bases, parts[index])
		}
	}
	// FROM命令がない場合は、根拠のないnodeを作りません。
	if len(bases) == 0 {
		return emptyDeploymentTopology()
	}

	// 同じbase imageが複数stageに現れても、nodeとrelationは一件ずつ作ります。
	runtimeID := "docker:" + sourceName
	uniqueBases := uniqueStrings(bases)
	topology := DeploymentTopology{Nodes: []DeploymentNode{{
		ID: runtimeID, Label: sourceName, Kind: "container-image", Confidence: ConfidenceConfirmed,
		Environment: "container", Source: sourceName, Metadata: map[string]string{"base_images": strings.Join(bases, ", ")},
	}}}
	for _, base := range uniqueBases {
		topology.Nodes = append(topology.Nodes, DeploymentNode{
			ID: "image:" + base, Label: base, Kind: "external-image", Confidence: ConfidenceConfirmed,
			Environment: "registry", Source: sourceName,
		})
		topology.Connections = append(topology.Connections, DeploymentConnection{
			Source: runtimeID, Target: "image:" + base, Relation: "FROM", Confidence: ConfidenceConfirmed,
		})
	}
	return sortDeploymentTopology(topology)
}

//	{
//	  責務: [
//	    ParseCompose: Compose service定義を確定deployment factsへ変換します
//	  ]
//	  処理: [
//	    1: YAML documentを検証する
//	    2: service nodeを作成する
//	    3: 宣言されたservice dependencyを接続する
//	    4: 安定順に返す
//	  ]
//	  引数: [
//	    source: Compose YAML本文
//	    sourceName: deployment設定のproject相対path
//	  ]
//	  戻り値: [
//	    DeploymentTopology: 抽出したfacts
//	    error: YAMLが不正な場合の解析error
//	  ]
//	}
func ParseCompose(source, sourceName string) (DeploymentTopology, error) {
	documents, err := decodeYAMLDocuments(source, sourceName)
	if err != nil {
		return DeploymentTopology{}, err
	}
	if len(documents) == 0 {
		return emptyDeploymentTopology(), nil
	}
	services, ok := documents[0]["services"].(map[string]any)
	if !ok {
		return emptyDeploymentTopology(), nil
	}
	// Dependency edgeは定義済みserviceだけへ接続します。
	known := make(map[string]struct{}, len(services))
	for name := range services {
		known[name] = struct{}{}
	}
	topology := emptyDeploymentTopology()
	// Service metadataは表示に必要な値だけを保持し、Compose構文を漏らしません。
	for name, value := range services {
		spec, _ := value.(map[string]any)
		image := ""
		if spec["image"] != nil {
			image = yamlValueString(spec["image"])
		}
		metadata := make(map[string]string)
		for key, value := range map[string]any{"image": spec["image"], "build": spec["build"]} {
			if value != nil {
				metadata[key] = yamlValueString(value)
			}
		}
		if ports := yamlStringList(spec["ports"]); len(ports) > 0 {
			metadata["ports"] = strings.Join(ports, ", ")
		}
		if len(metadata) == 0 {
			metadata = nil
		}
		topology.Nodes = append(topology.Nodes, DeploymentNode{
			ID: "compose:" + name, Label: name, Kind: classifyDeploymentService(name, image),
			Confidence: ConfidenceConfirmed, Environment: "compose", Source: sourceName, Metadata: metadata,
		})
		for _, dependency := range yamlStringList(spec["depends_on"]) {
			if _, exists := known[dependency]; exists {
				topology.Connections = append(topology.Connections, DeploymentConnection{
					Source: "compose:" + name, Target: "compose:" + dependency,
					Relation: "depends_on", Confidence: ConfidenceConfirmed,
				})
			}
		}
		for _, link := range yamlStringList(spec["links"]) {
			dependency, _, _ := strings.Cut(link, ":")
			if _, exists := known[dependency]; exists {
				topology.Connections = append(topology.Connections, DeploymentConnection{
					Source: "compose:" + name, Target: "compose:" + dependency,
					Relation: "links", Confidence: ConfidenceConfirmed,
				})
			}
		}
	}
	return sortDeploymentTopology(topology), nil
}
