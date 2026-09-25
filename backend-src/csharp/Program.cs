using System.Text.Json;
using Tomiya.CodeAtlas.CSharpBackend;

const string ContractVersion = "1";
const string BackendId = "csharp-roslyn-helper";

JsonSerializerOptions jsonOptions = new()
{
    PropertyNameCaseInsensitive = true,
};

try
{
    string input = await Console.In.ReadToEndAsync();
    BackendRequest? request = JsonSerializer.Deserialize<BackendRequest>(input, jsonOptions);

    if (request is null)
    {
        await WriteError("", "protocol_error", "Request body is empty or invalid.");
        return;
    }

    if (
        request.ContractVersion != ContractVersion
        || request.Operation != "parse"
        || request.Language != "csharp"
    )
    {
        await WriteError(request.RequestId, "protocol_error", "Unsupported parser backend request.");
        return;
    }

    RoslynAnalyzer analyzer = new();
    ModuleDto module = analyzer.Analyze(request.Source, request.Path ?? "<source.cs>");
    await WriteResponse(new BackendResponse
    {
        RequestId = request.RequestId,
        Ok = true,
        Ir = module,
    });
}
catch (JsonException error)
{
    await WriteError("", "protocol_error", error.Message);
}
catch (Exception error)
{
    Console.Error.WriteLine(error);
    await WriteError("", "failure", error.Message);
}

async Task WriteError(string requestId, string kind, string message)
{
    await WriteResponse(new BackendResponse
    {
        RequestId = requestId,
        Ok = false,
        Error = new ErrorDto
        {
            Kind = kind,
            Message = message,
            BackendId = BackendId,
            Retryable = false,
        },
    });
}

async Task WriteResponse(BackendResponse response)
{
    string payload = JsonSerializer.Serialize(response, jsonOptions);
    await Console.Out.WriteLineAsync(payload);
}
