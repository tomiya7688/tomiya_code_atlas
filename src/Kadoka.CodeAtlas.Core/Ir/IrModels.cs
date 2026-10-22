namespace Kadoka.CodeAtlas.Core.Ir;

public enum IrNodeKind
{
    Module,
    Namespace,
    Package,
    Class,
    Struct,
    Interface,
    Enum,
    Function,
    Method,
    Constructor,
    Field,
    Property,
    Variable,
    Parameter,
    Condition,
    Loop,
    Return,
    ExceptionHandling,
    FunctionCall,
    ObjectConstruction,
}

public enum Visibility
{
    Unspecified,
    Public,
    Protected,
    Internal,
    Private,
}

public sealed class SourceLocation
{
    public string? Path { get; init; }
    public int StartLine { get; init; }
    public int StartColumn { get; init; }
    public int EndLine { get; init; }
    public int EndColumn { get; init; }
}

public sealed class IrReference
{
    public required string Kind { get; init; }
    public required string Target { get; init; }
    public string Confidence { get; init; } = "certain";
    public SourceLocation? Location { get; init; }
}

public sealed class IrNode
{
    public required IrNodeKind Kind { get; init; }
    public required string Name { get; init; }
    public string? QualifiedName { get; init; }
    public string? ParentQualifiedName { get; init; }
    public Visibility Visibility { get; init; }
    public IReadOnlyList<string> Modifiers { get; init; } = Array.Empty<string>();
    public IReadOnlyList<string> TypeNames { get; init; } = Array.Empty<string>();
    public IReadOnlyList<IrReference> References { get; init; } = Array.Empty<IrReference>();
    public SourceLocation? Location { get; init; }
    public IReadOnlyDictionary<string, string> Metadata { get; init; } = new Dictionary<string, string>();
}

public sealed class ModuleIr
{
    public required string Language { get; init; }
    public string? Path { get; init; }
    public IReadOnlyList<IrNode> Nodes { get; init; } = Array.Empty<IrNode>();
}
