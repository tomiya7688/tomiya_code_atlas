namespace Kadoka.CodeAtlas.Core.Contracts;

public interface IRenderer<in TModel>
{
    string FormatId { get; }

    string Render(TModel model);
}
