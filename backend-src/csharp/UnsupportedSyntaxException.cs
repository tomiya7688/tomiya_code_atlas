namespace Tomiya.CodeAtlas.CSharpBackend;

internal sealed class UnsupportedSyntaxException(string message, int? line) : Exception(message)
{
    public int? Line { get; } = line;
}
