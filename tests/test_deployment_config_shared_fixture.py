import json
from pathlib import Path

from Src.analyzers.deployment import parse_compose, parse_dockerfile, parse_kubernetes


FIXTURE = Path(__file__).parent / "fixtures" / "analyzers" / "deployment_config_v1.json"


# {
#   責務: [
#     _topology_as_json: Python topology modelを共通fixtureのJSON shapeへ変換します
#   ]
#   処理: [
#     1: node fieldを正規化する
#     2: connection fieldを正規化する
#     3: nodeとedgeを安定順に並べる
#   ]
#   引数: [
#     topology: Python deployment topology
#   ]
#   戻り値: [
#     dict: JSON fixtureと比較できるtopology
#   ]
# }
def _topology_as_json(topology):
    nodes = []
    for node in topology.nodes:
        value = {
            "id": node.id,
            "label": node.label,
            "kind": node.kind,
            "confidence": node.confidence,
        }
        if node.environment:
            value["environment"] = node.environment
        if node.source:
            value["source"] = node.source
        if node.metadata:
            value["metadata"] = dict(node.metadata)
        nodes.append(value)
    connections = [
        {
            "source": edge.source,
            "target": edge.target,
            "relation": edge.relation,
            "confidence": edge.confidence,
        }
        for edge in topology.connections
    ]
    nodes.sort(key=lambda node: node["id"])
    connections.sort(key=lambda edge: (edge["source"], edge["target"], edge["relation"]))
    return {"nodes": nodes, "connections": connections}


# {
#   責務: [
#     test_deployment_config_extractors_match_shared_golden: Pythonの3種類のconfig extractorを共有goldenと照合します
#   ]
#   処理: [
#     1: shared fixtureを読む
#     2: 各config extractorを呼び出す
#     3: 出力をgoldenと比較する
#   ]
#   引数: [
#     なし
#   ]
#   戻り値: [
#     なし: pytest assertionで一致を検証する
#   ]
# }
def test_deployment_config_extractors_match_shared_golden():
    fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
    for name, parser in (
        ("dockerfile", lambda item: parse_dockerfile(item["source"], source_name=item["source_name"])),
        ("compose", lambda item: parse_compose(item["source"], source_name=item["source_name"])),
        ("kubernetes", lambda item: parse_kubernetes(item["source"], source_name=item["source_name"])),
    ):
        item = fixture[name]
        assert _topology_as_json(parser(item)) == item["expected"]
