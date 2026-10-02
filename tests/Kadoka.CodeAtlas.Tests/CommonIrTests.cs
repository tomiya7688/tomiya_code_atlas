using Kadoka.CodeAtlas.Core.Ir;
using Xunit;

namespace Kadoka.CodeAtlas.Tests;

public sealed class CommonIrTests
{
    [Fact]
    public void CommonIrStoresNormalizedDataWithoutBehavior()
    {
        var module = new ModuleIr
        {
            Language = "python",
            Path = "sample.py",
            Nodes = new[]
            {
                new IrNode
                {
                    Kind = IrNodeKind.Function,
                    Name = "main",
                    QualifiedName = "main",
                    Location = new SourceLocation
                    {
                        Path = "sample.py",
                        StartLine = 1,
                        StartColumn = 0,
                        EndLine = 2,
                        EndColumn = 8,
                    },
                    References = new[]
                    {
                        new IrReference
                        {
                            Kind = "call",
                            Target = "print",
                        },
                    },
                },
            },
        };

        Assert.Equal("python", module.Language);
        Assert.Single(module.Nodes);
        Assert.Equal("main", module.Nodes[0].Name);
        Assert.Equal("print", module.Nodes[0].References[0].Target);
    }
}
