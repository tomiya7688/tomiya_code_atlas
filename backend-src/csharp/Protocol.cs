using System.Text.Json.Serialization;

namespace Tomiya.CodeAtlas.CSharpBackend;

internal sealed class BackendRequest
{
    [JsonPropertyName("contract_version")]
    public string ContractVersion { get; init; } = "";

    [JsonPropertyName("request_id")]
    public string RequestId { get; init; } = "";

    [JsonPropertyName("operation")]
    public string Operation { get; init; } = "";

    [JsonPropertyName("language")]
    public string Language { get; init; } = "";

    [JsonPropertyName("source")]
    public string Source { get; init; } = "";

    [JsonPropertyName("path")]
    public string? Path { get; init; }
}

internal sealed class BackendResponse
{
    [JsonPropertyName("contract_version")]
    public string ContractVersion { get; init; } = "1";

    [JsonPropertyName("request_id")]
    public string RequestId { get; init; } = "";

    [JsonPropertyName("ok")]
    public bool Ok { get; init; }

    [JsonPropertyName("ir")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public ModuleDto? Ir { get; init; }

    [JsonPropertyName("error")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public ErrorDto? Error { get; init; }
}

internal sealed class ErrorDto
{
    [JsonPropertyName("kind")]
    public string Kind { get; init; } = "failure";

    [JsonPropertyName("message")]
    public string Message { get; init; } = "";

    [JsonPropertyName("backend_id")]
    public string BackendId { get; init; } = "csharp-roslyn-helper";

    [JsonPropertyName("retryable")]
    public bool Retryable { get; init; }
}

internal sealed class ModuleDto
{
    [JsonPropertyName("language")]
    public string Language { get; init; } = "csharp";

    [JsonPropertyName("entities")]
    public List<EntityDto> Entities { get; init; } = [];

    [JsonPropertyName("imports")]
    public List<string> Imports { get; init; } = [];

    [JsonPropertyName("diagnostics")]
    public List<DiagnosticDto> Diagnostics { get; init; } = [];
}

internal sealed class EntityDto
{
    [JsonPropertyName("kind")]
    public string Kind { get; init; } = "";

    [JsonPropertyName("name")]
    public string Name { get; init; } = "";

    [JsonPropertyName("line")]
    public int Line { get; init; }

    [JsonPropertyName("end_line")]
    public int EndLine { get; init; }

    [JsonPropertyName("parent")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? Parent { get; init; }

    [JsonPropertyName("parameters")]
    public List<string> Parameters { get; init; } = [];

    [JsonPropertyName("calls")]
    public List<string> Calls { get; init; } = [];

    [JsonPropertyName("call_sequence")]
    public List<string> CallSequence { get; init; } = [];

    [JsonPropertyName("visibility")]
    public string Visibility { get; init; } = "unspecified";

    [JsonPropertyName("bases")]
    public List<string> Bases { get; init; } = [];

    [JsonPropertyName("declaration_kind")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? DeclarationKind { get; init; }

    [JsonPropertyName("type_parameters")]
    public List<string> TypeParameters { get; init; } = [];

    [JsonPropertyName("type_constraints")]
    public List<string> TypeConstraints { get; init; } = [];

    [JsonPropertyName("resolved_calls")]
    public List<string> ResolvedCalls { get; init; } = [];

    [JsonPropertyName("symbol_id")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? SymbolId { get; init; }

    [JsonPropertyName("is_async")]
    public bool IsAsync { get; init; }
}

internal sealed class DiagnosticDto
{
    [JsonPropertyName("kind")]
    public string Kind { get; init; } = "info";

    [JsonPropertyName("message")]
    public string Message { get; init; } = "";

    [JsonPropertyName("line")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public int? Line { get; init; }
}
