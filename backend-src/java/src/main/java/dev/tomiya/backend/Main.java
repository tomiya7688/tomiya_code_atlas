package dev.tomiya.backend;

import com.github.javaparser.JavaParser;
import com.github.javaparser.ParserConfiguration;
import com.github.javaparser.ParseResult;
import com.github.javaparser.Problem;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.Modifier;
import com.github.javaparser.ast.Node;
import com.github.javaparser.ast.body.ClassOrInterfaceDeclaration;
import com.github.javaparser.ast.body.AnnotationMemberDeclaration;
import com.github.javaparser.ast.body.CompactConstructorDeclaration;
import com.github.javaparser.ast.body.ConstructorDeclaration;
import com.github.javaparser.ast.body.EnumDeclaration;
import com.github.javaparser.ast.body.FieldDeclaration;
import com.github.javaparser.ast.body.MethodDeclaration;
import com.github.javaparser.ast.body.RecordDeclaration;
import com.github.javaparser.ast.body.TypeDeclaration;
import com.github.javaparser.ast.body.VariableDeclarator;
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
                || request.requestId == null || request.requestId.isBlank()
                || request.source == null
                || !"parse".equals(request.operation) || !"java".equals(request.language)) {
            write(Response.error(request == null ? "" : request.requestId, "protocol_error", "Unsupported parser backend request."));
            return;
        }

        try {
            ModuleDto module = analyze(request);
            write(Response.ok(request.requestId, module));
        } catch (UnsupportedSyntaxException error) {
            write(Response.error(request.requestId, "unsupported_syntax", error.getMessage()));
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
                .setLanguageLevel(ParserConfiguration.LanguageLevel.JAVA_26)
                .setSymbolResolver(new JavaSymbolSolver(typeSolver));
        JavaParser parser = new JavaParser(configuration);
        ParseResult<CompilationUnit> result = parser.parse(request.source);
        if (!result.getProblems().isEmpty()) {
            Problem problem = result.getProblems().get(0);
            String message = problem.getMessage();
            if (problem.getLocation().isPresent()) {
                int problemLine = problem.getLocation().get().getBegin().getRange()
                        .map(range -> range.begin.line).orElse(1);
                message = "line " + problemLine + ": " + message;
            }
            throw new UnsupportedSyntaxException(message);
        }
        if (result.getResult().isEmpty()) {
            throw new UnsupportedSyntaxException("JavaParser did not produce a complete compilation unit.");
        }

        ModuleDto module = new ModuleDto();
        CompilationUnit unit = result.getResult().get();
        unit.getPackageDeclaration().ifPresent(pkg ->
                module.entities.add(EntityDto.module(pkg.getNameAsString(), line(pkg), endLine(pkg))));
        unit.getImports().forEach(importDeclaration -> {
            String importedName = importDeclaration.getNameAsString();
            if (importDeclaration.isStatic() && !importDeclaration.isAsterisk()) {
                int memberSeparator = importedName.lastIndexOf('.');
                if (memberSeparator > 0) importedName = importedName.substring(0, memberSeparator);
            }
            module.imports.add(importedName + (importDeclaration.isAsterisk() ? ".*" : ""));
        });
        unit.getTypes().forEach(declaration -> addType(declaration, null, module));
        return module;
    }

    private static void addType(TypeDeclaration<?> declaration, String parent, ModuleDto module) {
        EntityDto type = EntityDto.type(declaration);
        type.parent = parent;
        module.entities.add(type);
        String qualifiedParent = parent == null ? declaration.getNameAsString()
                : parent + "." + declaration.getNameAsString();

        if (declaration instanceof RecordDeclaration record) {
            record.getParameters().forEach(parameter -> {
                EntityDto component = EntityDto.field(parameter.getNameAsString(), parameter.getType().asString(),
                        parameter, parameter, "record_component");
                component.parent = qualifiedParent;
                module.entities.add(component);
            });
        }
        if (declaration instanceof EnumDeclaration enumDeclaration) {
            enumDeclaration.getEntries().forEach(constant -> {
                EntityDto entity = EntityDto.field(constant.getNameAsString(), declaration.getNameAsString(),
                        constant, declaration, "enum_constant");
                entity.parent = qualifiedParent;
                module.entities.add(entity);
            });
        }

        for (var member : declaration.getMembers()) {
            if (member instanceof TypeDeclaration<?> nested) {
                addType(nested, qualifiedParent, module);
            } else if (member instanceof FieldDeclaration field) {
                for (VariableDeclarator variable : field.getVariables()) {
                    EntityDto entity = EntityDto.field(variable.getNameAsString(), variable.getType().asString(),
                            variable, field, "field");
                    entity.parent = qualifiedParent;
                    module.entities.add(entity);
                }
            } else if (member instanceof MethodDeclaration method) {
                EntityDto entity = EntityDto.method(method);
                entity.parent = qualifiedParent;
                addCalls(method, method, entity, module);
                module.entities.add(entity);
            } else if (member instanceof ConstructorDeclaration constructor) {
                EntityDto entity = EntityDto.constructor(constructor);
                entity.parent = qualifiedParent;
                addCalls(constructor, constructor, entity, module);
                module.entities.add(entity);
            } else if (member instanceof CompactConstructorDeclaration constructor
                    && declaration instanceof RecordDeclaration record) {
                EntityDto entity = EntityDto.compactConstructor(constructor, record);
                entity.parent = qualifiedParent;
                addCalls(constructor, constructor, entity, module);
                module.entities.add(entity);
            } else if (member instanceof AnnotationMemberDeclaration annotationMember) {
                EntityDto entity = EntityDto.annotationMember(annotationMember);
                entity.parent = qualifiedParent;
                module.entities.add(entity);
            }
        }
    }

    private static void addCalls(Node owner, Node callable, EntityDto entity, ModuleDto module) {
        for (MethodCallExpr call : owner.findAll(MethodCallExpr.class)) {
            if (!belongsToCallable(call, callable)) continue;
            entity.calls.add(call.getNameAsString());
            entity.callSequence.add(call.getNameAsString());
            try {
                entity.resolvedCalls.add(call.resolve().getQualifiedSignature());
            } catch (RuntimeException error) {
                module.diagnostics.add(new DiagnosticDto(
                        "unresolved_symbol", call + ": " + compact(error), line(call)));
            }
        }
    }

    private static boolean belongsToCallable(MethodCallExpr call, Node expected) {
        for (Node current = call; current != null; current = current.getParentNode().orElse(null)) {
            if (current instanceof MethodDeclaration || current instanceof ConstructorDeclaration
                    || current instanceof CompactConstructorDeclaration) {
                return current == expected;
            }
        }
        return false;
    }

    private static final class UnsupportedSyntaxException extends RuntimeException {
        UnsupportedSyntaxException(String message) { super(message); }
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
        @SerializedName("parameter_types") List<List<String>> parameterTypes = new ArrayList<>();
        @SerializedName("resolved_calls") List<String> resolvedCalls = new ArrayList<>();
        @SerializedName("symbol_id") String symbolId;
        @SerializedName("is_async") boolean isAsync;
        @SerializedName("return_type") String returnType;
        @SerializedName("type_name") String typeName;
        String docstring;

        static EntityDto module(String name, int line, int endLine) {
            EntityDto dto = new EntityDto();
            dto.kind = "module";
            dto.name = name;
            dto.line = line;
            dto.endLine = endLine;
            dto.declarationKind = "package";
            return dto;
        }

        static EntityDto type(TypeDeclaration<?> declaration) {
            EntityDto dto = new EntityDto();
            dto.kind = "class";
            dto.name = declaration.getNameAsString();
            dto.line = line(declaration);
            dto.endLine = endLine(declaration);
            dto.visibility = visibility(declaration);
            dto.declarationKind = declarationKind(declaration);
            if (declaration instanceof ClassOrInterfaceDeclaration classOrInterface) {
                classOrInterface.getExtendedTypes().forEach(type -> dto.bases.add(type.asString()));
                classOrInterface.getImplementedTypes().forEach(type -> dto.bases.add(type.asString()));
                classOrInterface.getTypeParameters().forEach(type ->
                        addTypeParameter(dto, type.getNameAsString(), type.toString()));
            } else if (declaration instanceof EnumDeclaration enumDeclaration) {
                enumDeclaration.getImplementedTypes().forEach(type -> dto.bases.add(type.asString()));
            } else if (declaration instanceof RecordDeclaration record) {
                record.getImplementedTypes().forEach(type -> dto.bases.add(type.asString()));
                record.getTypeParameters().forEach(type ->
                        addTypeParameter(dto, type.getNameAsString(), type.toString()));
            }
            declaration.getAnnotations().forEach(annotation -> dto.decorators.add(annotation.getNameAsString()));
            dto.docstring = declaration.getJavadocComment().map(comment -> comment.parse().toText()).orElse(null);
            try {
                dto.symbolId = declaration.resolve().getQualifiedName();
            } catch (RuntimeException ignored) {
                dto.symbolId = null;
            }
            return dto;
        }

        private static String declarationKind(TypeDeclaration<?> declaration) {
            if (declaration instanceof RecordDeclaration) return "record";
            if (declaration instanceof EnumDeclaration) return "enum";
            if (declaration instanceof com.github.javaparser.ast.body.AnnotationDeclaration) return "annotation";
            if (declaration instanceof ClassOrInterfaceDeclaration classOrInterface) {
                return classOrInterface.isInterface() ? "interface" : "class";
            }
            return "class";
        }

        static EntityDto field(String name, String fieldType, Node variable, Node declaration, String fieldKind) {
            EntityDto dto = new EntityDto();
            dto.kind = "field";
            dto.name = name;
            dto.line = line(variable);
            dto.endLine = endLine(variable);
            dto.visibility = visibility(declaration);
            dto.declarationKind = fieldKind;
            dto.typeName = fieldType;
            dto.docstring = declaration.getComment().map(comment -> comment.getContent()).orElse(null);
            if (declaration instanceof com.github.javaparser.ast.nodeTypes.NodeWithAnnotations<?> annotated) {
                annotated.getAnnotations().forEach(annotation -> dto.decorators.add(annotation.getNameAsString()));
            }
            return dto;
        }

        private static void addTypeParameter(EntityDto dto, String name, String declaration) {
            dto.typeParameters.add(name);
            if (!declaration.equals(name)) dto.typeConstraints.add(declaration);
        }

        static EntityDto method(MethodDeclaration method) {
            EntityDto dto = new EntityDto();
            dto.kind = "method";
            dto.name = method.getNameAsString();
            dto.line = line(method);
            dto.endLine = endLine(method);
            dto.visibility = visibility(method);
            boolean explicitlyAbstract = method.getModifiers().stream()
                    .anyMatch(modifier -> modifier.getKeyword() == Modifier.Keyword.ABSTRACT);
            boolean implicitInterfaceAbstract = method.getBody().isEmpty()
                    && method.findAncestor(ClassOrInterfaceDeclaration.class)
                    .map(ClassOrInterfaceDeclaration::isInterface).orElse(false)
                    && method.getModifiers().stream().noneMatch(modifier ->
                            modifier.getKeyword() == Modifier.Keyword.STATIC
                                    || modifier.getKeyword() == Modifier.Keyword.PRIVATE);
            dto.declarationKind = explicitlyAbstract || implicitInterfaceAbstract
                    ? "abstract_method" : "method";
            dto.returnType = method.getType().asString();
            method.getParameters().forEach(parameter ->
                    addParameter(dto, parameter.getNameAsString(), parameterType(parameter)));
            method.getTypeParameters().forEach(type -> {
                addTypeParameter(dto, type.getNameAsString(), type.toString());
            });
            method.getAnnotations().forEach(annotation -> dto.decorators.add(annotation.getNameAsString()));
            dto.docstring = method.getJavadocComment().map(comment -> comment.parse().toText()).orElse(null);
            dto.isAsync = method.getType().asString().contains("CompletableFuture");
            try {
                dto.symbolId = method.resolve().getQualifiedSignature();
            } catch (RuntimeException ignored) {
                dto.symbolId = null;
            }
            return dto;
        }

        static EntityDto constructor(ConstructorDeclaration constructor) {
            EntityDto dto = new EntityDto();
            dto.kind = "method";
            dto.name = constructor.getNameAsString();
            dto.line = line(constructor);
            dto.endLine = endLine(constructor);
            dto.visibility = visibility(constructor);
            dto.declarationKind = "constructor";
            constructor.getParameters().forEach(parameter ->
                    addParameter(dto, parameter.getNameAsString(), parameterType(parameter)));
            constructor.getTypeParameters().forEach(type ->
                    addTypeParameter(dto, type.getNameAsString(), type.toString()));
            constructor.getAnnotations().forEach(annotation -> dto.decorators.add(annotation.getNameAsString()));
            dto.docstring = constructor.getJavadocComment().map(comment -> comment.parse().toText()).orElse(null);
            try {
                dto.symbolId = constructor.resolve().getQualifiedSignature();
            } catch (RuntimeException ignored) {
                dto.symbolId = null;
            }
            return dto;
        }

        static EntityDto compactConstructor(CompactConstructorDeclaration constructor, RecordDeclaration record) {
            EntityDto dto = new EntityDto();
            dto.kind = "method";
            dto.name = constructor.getNameAsString();
            dto.line = line(constructor);
            dto.endLine = endLine(constructor);
            dto.visibility = visibility(constructor);
            dto.declarationKind = "constructor";
            record.getParameters().forEach(parameter ->
                    addParameter(dto, parameter.getNameAsString(), parameterType(parameter)));
            constructor.getTypeParameters().forEach(type ->
                    addTypeParameter(dto, type.getNameAsString(), type.toString()));
            dto.docstring = constructor.getJavadocComment().map(comment -> comment.parse().toText()).orElse(null);
            try {
                dto.symbolId = constructor.resolve().getQualifiedSignature();
            } catch (RuntimeException ignored) {
                dto.symbolId = null;
            }
            return dto;
        }

        static EntityDto annotationMember(AnnotationMemberDeclaration member) {
            EntityDto dto = new EntityDto();
            dto.kind = "method";
            dto.name = member.getNameAsString();
            dto.line = line(member);
            dto.endLine = endLine(member);
            dto.visibility = "public";
            dto.declarationKind = "annotation_member";
            dto.returnType = member.getType().asString();
            dto.docstring = member.getJavadocComment().map(comment -> comment.parse().toText()).orElse(null);
            member.getAnnotations().forEach(annotation -> dto.decorators.add(annotation.getNameAsString()));
            return dto;
        }

        private static void addParameter(EntityDto dto, String name, String type) {
            dto.parameters.add(name);
            dto.parameterTypes.add(List.of(name, type));
        }

        private static String parameterType(com.github.javaparser.ast.body.Parameter parameter) {
            return parameter.getType().asString() + (parameter.isVarArgs() ? "[]" : "");
        }
    }
}

