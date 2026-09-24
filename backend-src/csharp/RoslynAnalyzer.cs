using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;

namespace Tomiya.CodeAtlas.CSharpBackend;

internal sealed class RoslynAnalyzer
{
    public ModuleDto Analyze(string source, string path)
    {
        CSharpParseOptions parseOptions = new(
            languageVersion: LanguageVersion.Preview,
            documentationMode: DocumentationMode.Parse,
            kind: SourceCodeKind.Regular
        );
        SyntaxTree tree = CSharpSyntaxTree.ParseText(source, parseOptions, path);
        CompilationUnitSyntax root = tree.GetCompilationUnitRoot();

        CSharpCompilation compilation = CSharpCompilation.Create(
            assemblyName: "TomiyaAnalysis",
            syntaxTrees: [tree],
            references: FrameworkReferences(),
            options: new CSharpCompilationOptions(
                OutputKind.DynamicallyLinkedLibrary,
                allowUnsafe: true
            )
        );
        SemanticModel model = compilation.GetSemanticModel(tree, ignoreAccessibility: true);

        ModuleDto result = new();
        foreach (UsingDirectiveSyntax directive in root.Usings)
        {
            string? name = directive.Name?.ToString();
            if (!string.IsNullOrWhiteSpace(name) && !result.Imports.Contains(name))
            {
                result.Imports.Add(name);
            }
        }

        foreach (TypeDeclarationSyntax type in root.DescendantNodes().OfType<TypeDeclarationSyntax>())
        {
            result.Entities.Add(TypeEntity(type, tree, model));
        }

        foreach (MethodDeclarationSyntax method in root.DescendantNodes().OfType<MethodDeclarationSyntax>())
        {
            result.Entities.Add(MethodEntity(method, tree, model));
        }

        foreach (ConstructorDeclarationSyntax constructor in root.DescendantNodes().OfType<ConstructorDeclarationSyntax>())
        {
            result.Entities.Add(ConstructorEntity(constructor, tree, model));
        }

        foreach (LocalFunctionStatementSyntax local in root.DescendantNodes().OfType<LocalFunctionStatementSyntax>())
        {
            result.Entities.Add(LocalFunctionEntity(local, tree, model));
        }

        foreach (Diagnostic diagnostic in compilation.GetDiagnostics())
        {
            FileLinePositionSpan lineSpan = diagnostic.Location.GetLineSpan();
            int? line = diagnostic.Location.IsInSource
                ? lineSpan.StartLinePosition.Line + 1
                : null;
            result.Diagnostics.Add(new DiagnosticDto
            {
                Kind = diagnostic.Severity.ToString().ToLowerInvariant(),
                Message = diagnostic.GetMessage(),
                Line = line,
            });
        }

        return result;
    }

    private static EntityDto TypeEntity(
        TypeDeclarationSyntax type,
        SyntaxTree tree,
        SemanticModel model
    )
    {
        INamedTypeSymbol? symbol = model.GetDeclaredSymbol(type);
        string? parent = type.Ancestors().OfType<TypeDeclarationSyntax>().FirstOrDefault()?.Identifier.Text;
        List<string> bases = symbol is null
            ? type.BaseList?.Types.Select(item => item.Type.ToString()).ToList() ?? []
            : SemanticBases(symbol);

        return new EntityDto
        {
            Kind = "class",
            Name = type.Identifier.Text,
            Line = StartLine(tree, type),
            EndLine = EndLine(tree, type),
            Parent = parent,
            Visibility = Visibility(symbol?.DeclaredAccessibility),
            Bases = bases,
            DeclarationKind = DeclarationKind(type),
            TypeParameters = symbol?.TypeParameters.Select(item => item.Name).ToList()
                ?? type.TypeParameterList?.Parameters.Select(item => item.Identifier.Text).ToList()
                ?? [],
            TypeConstraints = type.ConstraintClauses.Select(item => item.ToString()).ToList(),
            SymbolId = SymbolId(symbol),
        };
    }

    private static EntityDto MethodEntity(
        MethodDeclarationSyntax method,
        SyntaxTree tree,
        SemanticModel model
    )
    {
        IMethodSymbol? symbol = model.GetDeclaredSymbol(method);
        string? parent = method.Ancestors().OfType<TypeDeclarationSyntax>().FirstOrDefault()?.Identifier.Text;
        (List<string> calls, List<string> resolved) = CallsFor(method, method, model);

        return new EntityDto
        {
            Kind = "method",
            Name = method.Identifier.Text,
            Line = StartLine(tree, method),
            EndLine = EndLine(tree, method),
            Parent = parent,
            Parameters = method.ParameterList.Parameters.Select(item => item.Identifier.Text).ToList(),
            Calls = calls.Distinct(StringComparer.Ordinal).ToList(),
            CallSequence = calls,
            Visibility = Visibility(symbol?.DeclaredAccessibility),
            DeclarationKind = "method",
            TypeParameters = symbol?.TypeParameters.Select(item => item.Name).ToList()
                ?? method.TypeParameterList?.Parameters.Select(item => item.Identifier.Text).ToList()
                ?? [],
            TypeConstraints = method.ConstraintClauses.Select(item => item.ToString()).ToList(),
            ResolvedCalls = resolved,
            SymbolId = SymbolId(symbol),
            IsAsync = method.Modifiers.Any(SyntaxKind.AsyncKeyword),
        };
    }

    private static EntityDto ConstructorEntity(
        ConstructorDeclarationSyntax constructor,
        SyntaxTree tree,
        SemanticModel model
    )
    {
        IMethodSymbol? symbol = model.GetDeclaredSymbol(constructor);
        string? parent = constructor.Ancestors().OfType<TypeDeclarationSyntax>().FirstOrDefault()?.Identifier.Text;
        (List<string> calls, List<string> resolved) = CallsFor(constructor, constructor, model);

        return new EntityDto
        {
            Kind = "method",
            Name = constructor.Identifier.Text,
            Line = StartLine(tree, constructor),
            EndLine = EndLine(tree, constructor),
            Parent = parent,
            Parameters = constructor.ParameterList.Parameters.Select(item => item.Identifier.Text).ToList(),
            Calls = calls.Distinct(StringComparer.Ordinal).ToList(),
            CallSequence = calls,
            Visibility = Visibility(symbol?.DeclaredAccessibility),
            DeclarationKind = "constructor",
            ResolvedCalls = resolved,
            SymbolId = SymbolId(symbol),
        };
    }

    private static EntityDto LocalFunctionEntity(
        LocalFunctionStatementSyntax local,
        SyntaxTree tree,
        SemanticModel model
    )
    {
        IMethodSymbol? symbol = model.GetDeclaredSymbol(local);
        MethodDeclarationSyntax? method = local.Ancestors().OfType<MethodDeclarationSyntax>().FirstOrDefault();
        TypeDeclarationSyntax? type = local.Ancestors().OfType<TypeDeclarationSyntax>().FirstOrDefault();
        string? parent = method is null
            ? type?.Identifier.Text
            : $"{type?.Identifier.Text}.{method.Identifier.Text}";
        (List<string> calls, List<string> resolved) = CallsFor(local, local, model);

        return new EntityDto
        {
            Kind = "function",
            Name = local.Identifier.Text,
            Line = StartLine(tree, local),
            EndLine = EndLine(tree, local),
            Parent = parent,
            Parameters = local.ParameterList.Parameters.Select(item => item.Identifier.Text).ToList(),
            Calls = calls.Distinct(StringComparer.Ordinal).ToList(),
            CallSequence = calls,
            Visibility = "private",
            DeclarationKind = "local_function",
            TypeParameters = symbol?.TypeParameters.Select(item => item.Name).ToList() ?? [],
            TypeConstraints = local.ConstraintClauses.Select(item => item.ToString()).ToList(),
            ResolvedCalls = resolved,
            SymbolId = SymbolId(symbol),
            IsAsync = local.Modifiers.Any(SyntaxKind.AsyncKeyword),
        };
    }

    private static (List<string> Calls, List<string> Resolved) CallsFor(
        SyntaxNode callable,
        SyntaxNode boundary,
        SemanticModel model
    )
    {
        List<string> calls = [];
        List<string> resolved = [];

        foreach (InvocationExpressionSyntax invocation in callable.DescendantNodes().OfType<InvocationExpressionSyntax>())
        {
            SyntaxNode? nearestCallable = invocation.Ancestors().FirstOrDefault(IsCallableBoundary);
            if (!ReferenceEquals(nearestCallable, boundary))
            {
                continue;
            }

            SymbolInfo symbolInfo = model.GetSymbolInfo(invocation);
            IMethodSymbol? symbol = symbolInfo.Symbol as IMethodSymbol;
            string syntaxName = InvocationName(invocation.Expression);
            string callName = symbol?.Name ?? syntaxName;
            if (!string.IsNullOrWhiteSpace(callName))
            {
                calls.Add(callName);
            }
            if (symbol is not null)
            {
                resolved.Add(symbol.ToDisplayString(SymbolDisplayFormat.CSharpErrorMessageFormat));
            }
        }

        return (calls, resolved);
    }

    private static bool IsCallableBoundary(SyntaxNode node) =>
        node is MethodDeclarationSyntax
        or ConstructorDeclarationSyntax
        or LocalFunctionStatementSyntax
        or AnonymousFunctionExpressionSyntax;

    private static string InvocationName(ExpressionSyntax expression) =>
        expression switch
        {
            IdentifierNameSyntax identifier => identifier.Identifier.Text,
            MemberAccessExpressionSyntax member => member.Name.Identifier.Text,
            GenericNameSyntax generic => generic.Identifier.Text,
            MemberBindingExpressionSyntax binding => binding.Name.Identifier.Text,
            _ => expression.ToString(),
        };

    private static List<MetadataReference> FrameworkReferences()
    {
        string? trusted = AppContext.GetData("TRUSTED_PLATFORM_ASSEMBLIES") as string;
        if (string.IsNullOrWhiteSpace(trusted))
        {
            return [];
        }

        return trusted
            .Split(Path.PathSeparator, StringSplitOptions.RemoveEmptyEntries)
            .Distinct(StringComparer.OrdinalIgnoreCase)
            .Select(path => (MetadataReference)MetadataReference.CreateFromFile(path))
            .ToList();
    }

    private static List<string> SemanticBases(INamedTypeSymbol symbol)
    {
        List<string> result = [];
        if (symbol.BaseType is { SpecialType: not SpecialType.System_Object } baseType)
        {
            result.Add(baseType.Name);
        }
        result.AddRange(symbol.Interfaces.Select(item => item.Name));
        return result.Distinct(StringComparer.Ordinal).ToList();
    }

    private static string DeclarationKind(TypeDeclarationSyntax type) =>
        type switch
        {
            InterfaceDeclarationSyntax => "interface",
            StructDeclarationSyntax => "struct",
            RecordDeclarationSyntax record when record.ClassOrStructKeyword.IsKind(SyntaxKind.StructKeyword) => "record_struct",
            RecordDeclarationSyntax => "record",
            _ => "class",
        };

    private static string Visibility(Accessibility? accessibility) =>
        accessibility switch
        {
            Accessibility.Public => "public",
            Accessibility.Protected => "protected",
            Accessibility.Internal => "internal",
            Accessibility.Private => "private",
            Accessibility.ProtectedAndInternal => "protected",
            Accessibility.ProtectedOrInternal => "protected",
            _ => "unspecified",
        };

    private static string? SymbolId(ISymbol? symbol) =>
        symbol?.ToDisplayString(SymbolDisplayFormat.FullyQualifiedFormat);

    private static int StartLine(SyntaxTree tree, SyntaxNode node) =>
        tree.GetLineSpan(node.Span).StartLinePosition.Line + 1;

    private static int EndLine(SyntaxTree tree, SyntaxNode node) =>
        tree.GetLineSpan(node.Span).EndLinePosition.Line + 1;
}
