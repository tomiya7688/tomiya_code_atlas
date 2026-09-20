package dev.kadoka.backend;

import com.github.javaparser.JavaParser;
import com.github.javaparser.ParserConfiguration;
import com.github.javaparser.ParseResult;
import com.github.javaparser.Problem;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.Modifier;
import com.github.javaparser.ast.Node;
import com.github.javaparser.ast.body.ClassOrInterfaceDeclaration;
import com.github.javaparser.ast.body.MethodDeclaration;
import com.github.javaparser.ast.expr.MethodCallExpr;
import com.github.javaparser.resolution.UnsolvedSymbolException;
import com.github.javaparser.symbolsolver.JavaSymbolSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.CombinedTypeSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.JavaParserTypeSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.ReflectionTypeSolver;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.annotations.SerializedName;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;

public final class Main {
    private static final String CONTRACT_VERSION = "1";
    private static final String BACKEND_ID = "java-javaparser-symbol-solver-helper";
    private static final Gson GSON = new GsonBuilder().disableHtmlEscaping().create();

    public static void main(String[] args) throws IOException {
        String input = new String(System.in.readAllBytes(), StandardCharsets.UTF_8);
        Request request;
        try {
            request = GSON.fromJson(input, Request.class);
        } catch (RuntimeException error) {
            write(Response.error("", "protocol_error", error.getMessage()));
            return;
        }
        if (request == null || !CONTRACT_VERSION.equals(request.contractVersion)
                || !"parse".equals(request.operation) || !"java".equals(request.language)) {
            write(Response.error(request == null ? "" : request.requestId, "protocol_error", "Unsupported parser backend request."));
            return;
        }

        try {
            ModuleDto module = analyze(request);
            write(Response.ok(request.requestId, module));
        } catch (Exception error) {
            System.err.println(error);
            write(Response.error(request.requestId, "failure", error.getMessage()));
        }
    }

    private static ModuleDto analyze(Request request) {
        CombinedTypeSolver typeSolver = new CombinedTypeSolver();
        typeSolver.add(new ReflectionTypeSolver());

        if (request.path != null && !request.path.isBlank()) {
            try {
                Path sourcePath = Path.of(request.path).toAbsolutePath().normalize();
                Path root = sourcePath.getParent();
                if (root != null && Files.isDirectory(root)) {
                    typeSolver.add(new JavaParserTypeSolver(root));
                }
            } catch (RuntimeException ignored) {
                // In-memory/editor paths are valid; unresolved symbols remain explicit diagnostics.
            }
        }

        ParserConfiguration configuration = new ParserConfiguration()
                .setSymbolResolver(new JavaSymbolSolver(typeSolver));
        JavaParser parser = new JavaParser(configuration);
        ParseResult<CompilationUnit> result = parser.parse(request.source == null ? "" : request.source);

        ModuleDto module = new ModuleDto();
        for (Problem problem : result.getProblems()) {
            module.diagnostics.add(new DiagnosticDto("parse_problem", problem.getMessage(), null));
        }
        if (result.getResult().isEmpty()) {
            return module;
        }

        CompilationUnit unit = result.getResult().get();
        unit.getPackageDeclaration().ifPresent(pkg ->
                module.entities.add(EntityDto.module(pkg.getNameAsString(), line(pkg), endLine(pkg))));
        unit.getImports().forEach(importDeclaration ->
                module.imports.add(importDeclaration.getNameAsString() + (importDeclaration.isAsterisk() ? ".*" : "")));

        for (ClassOrInterfaceDeclaration declaration : unit.findAll(ClassOrInterfaceDeclaration.class)) {
            if (declaration.findAncestor(ClassOrInterfaceDeclaration.class).isPresent()) {
                continue;
            }
            EntityDto entity = EntityDto.type(declaration);
            module.entities.add(entity);
        }

        for (MethodDeclaration method : unit.findAll(MethodDeclaration.class)) {
            EntityDto entity = EntityDto.method(method);
            for (MethodCallExpr call : method.findAll(MethodCallExpr.class)) {
                if (call.findAncestor(MethodDeclaration.class).orElse(null) != method) {
                    continue;
                }
                entity.calls.add(call.getNameAsString());
                entity.callSequence.add(call.getNameAsString());
                try {
                    entity.resolvedCalls.add(call.resolve().getQualifiedSignature());
                } catch (RuntimeException error) {
                    Integer unresolvedLine = line(call);
                    module.diagnostics.add(new DiagnosticDto(
                            "unresolved_symbol",
                            call + ": " + compact(error),
                            unresolvedLine
                    ));
                }
            }
            module.entities.add(entity);
        }
        return module;
    }

    private static String compact(RuntimeException error) {
        if (error instanceof UnsolvedSymbolException) {
            return error.getMessage();
        }
        String message = error.getMessage();
        return message == null || message.isBlank() ? error.getClass().getSimpleName() : message;
    }

    private static int line(Node node) {
        return node.getRange().map(range -> range.begin.line).orElse(1);
    }

    private static int endLine(Node node) {
        return node.getRange().map(range -> range.end.line).orElse(line(node));
    }

    private static String visibility(Node node) {
        if (node instanceof com.github.javaparser.ast.nodeTypes.NodeWithModifiers<?> withModifiers) {
            Set<Modifier.Keyword> modifiers = new LinkedHashSet<>();
            withModifiers.getModifiers().forEach(modifier -> modifiers.add(modifier.getKeyword()));
            if (modifiers.contains(Modifier.Keyword.PUBLIC)) return "public";
            if (modifiers.contains(Modifier.Keyword.PROTECTED)) return "protected";
            if (modifiers.contains(Modifier.Keyword.PRIVATE)) return "private";
        }
        return "unspecified";
    }

    private static void write(Response response) {
        System.out.println(GSON.toJson(response));
    }

    static final class Request {
        @SerializedName("contract_version") String contractVersion;
        @SerializedName("request_id") String requestId;
        String operation;
        String language;
        String source;
        String path;
    }

    static final class Response {
        @SerializedName("contract_version") String contractVersion = CONTRACT_VERSION;
        @SerializedName("request_id") String requestId;
        boolean ok;
        @SerializedName("ir") ModuleDto ir;
        ErrorDto error;

        static Response ok(String requestId, ModuleDto module) {
            Response response = new Response();
            response.requestId = requestId;
            response.ok = true;
            response.ir = module;
            return response;
        }

        static Response error(String requestId, String kind, String message) {
            Response response = new Response();
            response.requestId = requestId == null ? "" : requestId;
            response.ok = false;
            response.error = new ErrorDto(kind, message == null ? "Parser backend failed." : message);
            return response;
        }
    }

    static final class ErrorDto {
        String kind;
        String message;
        @SerializedName("backend_id") String backendId = BACKEND_ID;
        boolean retryable = false;

        ErrorDto(String kind, String message) {
            this.kind = kind;
            this.message = message;
        }
    }

    static final class ModuleDto {
        String language = "java";
        List<EntityDto> entities = new ArrayList<>();
        List<String> imports = new ArrayList<>();
        List<DiagnosticDto> diagnostics = new ArrayList<>();
    }

    static final class DiagnosticDto {
        String kind;
        String message;
        Integer line;

        DiagnosticDto(String kind, String message, Integer line) {
            this.kind = kind;
            this.message = message;
            this.line = line;
        }
    }

    static final class EntityDto {
        String kind;
        String name;
        int line;
        @SerializedName("end_line") int endLine;
        String parent;
        List<String> parameters = new ArrayList<>();
        List<String> decorators = new ArrayList<>();
        List<String> calls = new ArrayList<>();
        @SerializedName("call_sequence") List<String> callSequence = new ArrayList<>();
        String visibility = "unspecified";
        List<String> bases = new ArrayList<>();
        @SerializedName("declaration_kind") String declarationKind;
        @SerializedName("type_parameters") List<String> typeParameters = new ArrayList<>();
        @SerializedName("type_constraints") List<String> typeConstraints = new ArrayList<>();
        @SerializedName("resolved_calls") List<String> resolvedCalls = new ArrayList<>();
        @SerializedName("symbol_id") String symbolId;
        @SerializedName("is_async") boolean isAsync;

        static EntityDto module(String name, int line, int endLine) {
            EntityDto dto = new EntityDto();
            dto.kind = "module";
            dto.name = name;
            dto.line = line;
            dto.endLine = endLine;
            dto.declarationKind = "package";
            return dto;
        }

        static EntityDto type(ClassOrInterfaceDeclaration declaration) {
            EntityDto dto = new EntityDto();
            dto.kind = "class";
            dto.name = declaration.getNameAsString();
            dto.line = line(declaration);
            dto.endLine = endLine(declaration);
            dto.visibility = visibility(declaration);
            dto.declarationKind = declaration.isInterface() ? "interface" : "class";
            declaration.getExtendedTypes().forEach(type -> dto.bases.add(type.getNameAsString()));
            declaration.getImplementedTypes().forEach(type -> dto.bases.add(type.getNameAsString()));
            declaration.getTypeParameters().forEach(type -> {
                dto.typeParameters.add(type.getNameAsString());
                if (!type.getTypeBound().isEmpty()) {
                    dto.typeConstraints.add(type.toString());
                }
            });
            declaration.getAnnotations().forEach(annotation -> dto.decorators.add(annotation.getNameAsString()));
            try {
                dto.symbolId = declaration.resolve().getQualifiedName();
            } catch (RuntimeException ignored) {
                dto.symbolId = null;
            }
            return dto;
        }

        static EntityDto method(MethodDeclaration method) {
            EntityDto dto = new EntityDto();
            dto.kind = "method";
            dto.name = method.getNameAsString();
            dto.line = line(method);
            dto.endLine = endLine(method);
            dto.parent = method.findAncestor(ClassOrInterfaceDeclaration.class)
                    .map(ClassOrInterfaceDeclaration::getNameAsString).orElse(null);
            dto.visibility = visibility(method);
            dto.declarationKind = "method";
            method.getParameters().forEach(parameter -> dto.parameters.add(parameter.getNameAsString()));
            method.getTypeParameters().forEach(type -> {
                dto.typeParameters.add(type.getNameAsString());
                if (!type.getTypeBound().isEmpty()) dto.typeConstraints.add(type.toString());
            });
            method.getAnnotations().forEach(annotation -> dto.decorators.add(annotation.getNameAsString()));
            dto.isAsync = method.getType().asString().contains("CompletableFuture");
            try {
                dto.symbolId = method.resolve().getQualifiedSignature();
            } catch (RuntimeException ignored) {
                dto.symbolId = null;
            }
            return dto;
        }
    }
}
