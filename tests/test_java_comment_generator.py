from pathlib import Path
from time import perf_counter

from Src.generators import CommentGenerator
from Src.models import CommentTarget

def test_java_candidates():
    source = """package demo;
public record Report(String name) {}
public final class ReportStore {
    private String cachedName;
    public Report loadReport(String name) { return new Report(name); }
}
"""
    got = CommentGenerator().candidates(source, "java")
    assert [(x.target, x.name) for x in got] == [(CommentTarget.CLASS, "Report"), (CommentTarget.CLASS, "ReportStore"), (CommentTarget.FIELD, "cachedName"), (CommentTarget.METHOD, "loadReport")]

def test_java_idempotent():
    generator = CommentGenerator()
    source = "public class Ready {\n    public boolean isReady() { return true; }\n}\n"
    once = generator.generate(source, "java")
    assert generator.generate(once, "java") == once


def test_java_comment_generation_finishes_for_backend_source():
    source_path = (
        Path(__file__).resolve().parents[1]
        / "backend-src/java/src/main/java/dev/tomiya/backend/Main.java"
    )
    source = source_path.read_text(encoding="utf-8")

    started = perf_counter()
    result = CommentGenerator().generate(source, "java")

    assert perf_counter() - started < 2.0
    assert len(result) > len(source)
