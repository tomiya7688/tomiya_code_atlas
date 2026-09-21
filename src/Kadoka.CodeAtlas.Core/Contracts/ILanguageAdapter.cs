using Kadoka.CodeAtlas.Core.Ir;

namespace Kadoka.CodeAtlas.Core.Contracts;

public interface ILanguageAdapter
{
    string LanguageId { get; }

    ModuleIr Parse(string source, string? path = null);
}
