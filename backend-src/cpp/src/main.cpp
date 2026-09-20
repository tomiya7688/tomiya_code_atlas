#include <algorithm>
#include <cctype>
#include <iostream>
#include <memory>
#include <regex>
#include <sstream>
#include <string>
#include <utility>
#include <vector>

#include "clang/AST/ASTContext.h"
#include "clang/AST/Decl.h"
#include "clang/AST/DeclCXX.h"
#include "clang/AST/RecursiveASTVisitor.h"
#include "clang/Frontend/ASTUnit.h"
#include "clang/Tooling/CompilationDatabase.h"
#include "clang/Tooling/Tooling.h"
#include "llvm/ADT/StringRef.h"
#include "llvm/Support/JSON.h"
#include "llvm/Support/raw_ostream.h"

namespace {

constexpr llvm::StringLiteral kContractVersion = "1";
constexpr llvm::StringLiteral kBackendId = "cpp-clang-tooling-helper";

struct Entity {
  std::string kind;
  std::string name;
  unsigned line = 1;
  unsigned endLine = 1;
  std::string parent;
  std::vector<std::string> parameters;
  std::vector<std::string> decorators;
  std::vector<std::string> calls;
  std::vector<std::string> callSequence;
  std::string visibility = "unspecified";
  std::vector<std::string> bases;
  std::string declarationKind;
  std::vector<std::string> typeParameters;
  std::vector<std::string> typeConstraints;
  std::vector<std::string> resolvedCalls;
  std::string symbolId;
  bool isAsync = false;
};

struct Diagnostic {
  std::string kind;
  std::string message;
  int line = 0;
};

llvm::json::Array stringsToJson(const std::vector<std::string> &values) {
  llvm::json::Array out;
  for (const auto &value : values) {
    out.emplace_back(value);
  }
  return out;
}

llvm::json::Object entityToJson(const Entity &entity) {
  llvm::json::Object out;
  out["kind"] = entity.kind;
  out["name"] = entity.name;
  out["line"] = static_cast<int64_t>(entity.line);
  out["end_line"] = static_cast<int64_t>(entity.endLine);
  if (!entity.parent.empty()) {
    out["parent"] = entity.parent;
  }
  out["parameters"] = stringsToJson(entity.parameters);
  out["decorators"] = stringsToJson(entity.decorators);
  out["calls"] = stringsToJson(entity.calls);
  out["call_sequence"] = stringsToJson(entity.callSequence);
  out["visibility"] = entity.visibility;
  out["bases"] = stringsToJson(entity.bases);
  if (!entity.declarationKind.empty()) {
    out["declaration_kind"] = entity.declarationKind;
  }
  out["type_parameters"] = stringsToJson(entity.typeParameters);
  out["type_constraints"] = stringsToJson(entity.typeConstraints);
  out["resolved_calls"] = stringsToJson(entity.resolvedCalls);
  if (!entity.symbolId.empty()) {
    out["symbol_id"] = entity.symbolId;
  }
  out["is_async"] = entity.isAsync;
  return out;
}

llvm::json::Object diagnosticToJson(const Diagnostic &diagnostic) {
  llvm::json::Object out;
  out["kind"] = diagnostic.kind;
  out["message"] = diagnostic.message;
  if (diagnostic.line > 0) {
    out["line"] = static_cast<int64_t>(diagnostic.line);
  }
  return out;
}

std::string accessName(clang::AccessSpecifier access) {
  switch (access) {
    case clang::AS_public:
      return "public";
    case clang::AS_protected:
      return "protected";
    case clang::AS_private:
      return "private";
    default:
      return "unspecified";
  }
}

std::pair<unsigned, unsigned> sourceLines(
    const clang::SourceManager &sourceManager,
    clang::SourceRange range) {
  auto begin = sourceManager.getPresumedLoc(sourceManager.getSpellingLoc(range.getBegin()));
  auto end = sourceManager.getPresumedLoc(sourceManager.getSpellingLoc(range.getEnd()));
  unsigned beginLine = begin.isValid() ? begin.getLine() : 1;
  unsigned endLine = end.isValid() ? end.getLine() : beginLine;
  return {beginLine, endLine};
}

bool inMainFile(const clang::SourceManager &sourceManager, clang::SourceLocation location) {
  if (location.isInvalid()) {
    return false;
  }
  return sourceManager.isWrittenInMainFile(sourceManager.getSpellingLoc(location));
}

std::string parentName(const clang::DeclContext *context) {
  while (context != nullptr) {
    if (const auto *record = llvm::dyn_cast<clang::CXXRecordDecl>(context)) {
      if (!record->getNameAsString().empty()) {
        return record->getNameAsString();
      }
    }
    if (const auto *nameSpace = llvm::dyn_cast<clang::NamespaceDecl>(context)) {
      if (!nameSpace->getNameAsString().empty()) {
        return nameSpace->getNameAsString();
      }
    }
    context = context->getParent();
  }
  return {};
}

class Visitor : public clang::RecursiveASTVisitor<Visitor> {
 public:
  Visitor(clang::ASTContext &context, std::vector<Entity> &entities,
          std::vector<Diagnostic> &diagnostics)
      : context_(context),
        sourceManager_(context.getSourceManager()),
        entities_(entities),
        diagnostics_(diagnostics) {}

  bool VisitNamespaceDecl(clang::NamespaceDecl *decl) {
    if (decl->isImplicit() || decl->getName().empty() ||
        !inMainFile(sourceManager_, decl->getLocation())) {
      return true;
    }
    Entity entity;
    entity.kind = "module";
    entity.name = decl->getNameAsString();
    entity.declarationKind = "namespace";
    auto lines = sourceLines(sourceManager_, decl->getSourceRange());
    entity.line = lines.first;
    entity.endLine = lines.second;
    entity.parent = parentName(decl->getDeclContext());
    entity.symbolId = decl->getQualifiedNameAsString();
    entities_.push_back(std::move(entity));
    return true;
  }

  bool VisitCXXRecordDecl(clang::CXXRecordDecl *decl) {
    if (decl->isImplicit() || !decl->isThisDeclarationADefinition() ||
        decl->getName().empty() ||
        !inMainFile(sourceManager_, decl->getLocation())) {
      return true;
    }

    Entity entity;
    entity.kind = "class";
    entity.name = decl->getNameAsString();
    entity.declarationKind = decl->isStruct() ? "struct" : "class";
    entity.parent = parentName(decl->getDeclContext());
    entity.visibility = accessName(decl->getAccess());
    entity.symbolId = decl->getQualifiedNameAsString();
    auto lines = sourceLines(sourceManager_, decl->getSourceRange());
    entity.line = lines.first;
    entity.endLine = lines.second;

    for (const auto &base : decl->bases()) {
      entity.bases.push_back(base.getType().getAsString());
    }

    if (auto *classTemplate = decl->getDescribedClassTemplate()) {
      for (const auto *parameter : *classTemplate->getTemplateParameters()) {
        if (const auto *named = llvm::dyn_cast<clang::NamedDecl>(parameter)) {
          auto name = named->getNameAsString();
          if (!name.empty()) {
            entity.typeParameters.push_back(name);
          }
        }
      }
    }

    entities_.push_back(std::move(entity));
    return true;
  }

  bool VisitEnumDecl(clang::EnumDecl *decl) {
    if (decl->isImplicit() || !decl->isThisDeclarationADefinition() ||
        decl->getName().empty() ||
        !inMainFile(sourceManager_, decl->getLocation())) {
      return true;
    }
    Entity entity;
    entity.kind = "class";
    entity.name = decl->getNameAsString();
    entity.declarationKind = "enum";
    entity.parent = parentName(decl->getDeclContext());
    entity.symbolId = decl->getQualifiedNameAsString();
    auto lines = sourceLines(sourceManager_, decl->getSourceRange());
    entity.line = lines.first;
    entity.endLine = lines.second;
    entities_.push_back(std::move(entity));
    return true;
  }

  bool TraverseFunctionDecl(clang::FunctionDecl *decl) {
    if (decl == nullptr) {
      return true;
    }
    if (llvm::isa<clang::CXXMethodDecl>(decl)) {
      return clang::RecursiveASTVisitor<Visitor>::TraverseFunctionDecl(decl);
    }
    return traverseCallable(decl, "function", "function", clang::AS_none, [&]() {
      return clang::RecursiveASTVisitor<Visitor>::TraverseFunctionDecl(decl);
    });
  }

  bool TraverseCXXMethodDecl(clang::CXXMethodDecl *decl) {
    if (decl == nullptr) {
      return true;
    }
    return traverseCallable(decl, "method", "method", decl->getAccess(), [&]() {
      return clang::RecursiveASTVisitor<Visitor>::TraverseCXXMethodDecl(decl);
    });
  }

  bool VisitCallExpr(clang::CallExpr *call) {
    if (functionStack_.empty() ||
        !inMainFile(sourceManager_, call->getExprLoc())) {
      return true;
    }

    auto &entity = entities_[static_cast<size_t>(functionStack_.back())];
    const clang::FunctionDecl *callee = call->getDirectCallee();
    std::string callName;
    if (callee != nullptr) {
      callName = callee->getNameAsString();
      entity.resolvedCalls.push_back(
          callee->getQualifiedNameAsString() + " " + callee->getType().getAsString());
    } else {
      llvm::raw_string_ostream stream(callName);
      call->getCallee()->printPretty(stream, nullptr, context_.getPrintingPolicy());
      stream.flush();
      if (callName.empty()) {
        callName = "<unresolved-call>";
      }
      Diagnostic diagnostic;
      diagnostic.kind = "unresolved_symbol";
      diagnostic.message = "Unable to bind C++ call target: " + callName;
      auto location = sourceManager_.getPresumedLoc(call->getExprLoc());
      diagnostic.line = location.isValid() ? static_cast<int>(location.getLine()) : 0;
      diagnostics_.push_back(std::move(diagnostic));
    }

    entity.callSequence.push_back(callName);
    if (std::find(entity.calls.begin(), entity.calls.end(), callName) == entity.calls.end()) {
      entity.calls.push_back(callName);
    }
    return true;
  }

 private:
  template <typename CallableDecl, typename TraverseFn>
  bool traverseCallable(CallableDecl *decl, llvm::StringRef kind,
                        llvm::StringRef declarationKind,
                        clang::AccessSpecifier access, TraverseFn traverse) {
    const bool shouldCapture =
        !decl->isImplicit() && decl->doesThisDeclarationHaveABody() &&
        inMainFile(sourceManager_, decl->getLocation());

    if (!shouldCapture) {
      return traverse();
    }

    Entity entity;
    entity.kind = kind.str();
    entity.name = decl->getNameAsString();
    entity.declarationKind = declarationKind.str();
    entity.parent = parentName(decl->getDeclContext());
    entity.symbolId =
        decl->getQualifiedNameAsString() + " " + decl->getType().getAsString();
    entity.visibility = accessName(access);
    auto lines = sourceLines(sourceManager_, decl->getSourceRange());
    entity.line = lines.first;
    entity.endLine = lines.second;

    for (const auto *parameter : decl->parameters()) {
      std::string rendered = parameter->getType().getAsString();
      if (!parameter->getName().empty()) {
        rendered += " " + parameter->getNameAsString();
      }
      entity.parameters.push_back(std::move(rendered));
    }

    if (const auto *functionTemplate = decl->getDescribedFunctionTemplate()) {
      for (const auto *parameter : *functionTemplate->getTemplateParameters()) {
        if (const auto *named = llvm::dyn_cast<clang::NamedDecl>(parameter)) {
          auto name = named->getNameAsString();
          if (!name.empty()) {
            entity.typeParameters.push_back(name);
          }
        }
      }
    }

    const auto returnType = decl->getReturnType().getAsString();
    entity.isAsync =
        returnType.find("future") != std::string::npos ||
        returnType.find("task") != std::string::npos;

    entities_.push_back(std::move(entity));
    functionStack_.push_back(static_cast<int>(entities_.size() - 1));
    const bool result = traverse();
    functionStack_.pop_back();
    return result;
  }

  clang::ASTContext &context_;
  clang::SourceManager &sourceManager_;
  std::vector<Entity> &entities_;
  std::vector<Diagnostic> &diagnostics_;
  std::vector<int> functionStack_;
};

void collectTextualPreprocessorFacts(const std::string &source,
                                      std::vector<std::string> &imports,
                                      std::vector<Entity> &entities) {
  std::regex includePattern(R"(^\s*#\s*include\s*[<"]([^>"]+)[>"])");
  std::regex macroPattern(R"(^\s*#\s*define\s+([A-Za-z_][A-Za-z0-9_]*))");
  std::istringstream stream(source);
  std::string line;
  unsigned lineNumber = 0;
  while (std::getline(stream, line)) {
    ++lineNumber;
    std::smatch match;
    if (std::regex_search(line, match, includePattern)) {
      imports.push_back(match[1].str());
    }
    if (std::regex_search(line, match, macroPattern)) {
      Entity macro;
      macro.kind = "module";
      macro.name = match[1].str();
      macro.line = lineNumber;
      macro.endLine = lineNumber;
      macro.declarationKind = "macro";
      macro.symbolId = "macro:" + macro.name;
      entities.push_back(std::move(macro));
    }
  }
}

std::vector<std::string> fallbackArgs() {
  return {"-xc++", "-std=c++20"};
}

std::vector<std::string> compileArgsForPath(const std::string &path,
                                            std::vector<Diagnostic> &diagnostics) {
  if (path.empty() || path.front() == '<') {
    return fallbackArgs();
  }

  std::string errorMessage;
  auto database = clang::tooling::CompilationDatabase::autoDetectFromSource(path, errorMessage);
  if (!database) {
    if (!errorMessage.empty()) {
      diagnostics.push_back(
          {"compile_commands_unavailable", errorMessage, 0});
    }
    return fallbackArgs();
  }

  auto commands = database->getCompileCommands(path);
  if (commands.empty()) {
    diagnostics.push_back(
        {"compile_commands_unavailable",
         "compile_commands.json was discovered but has no command for this source file.",
         0});
    return fallbackArgs();
  }

  std::vector<std::string> args;
  const auto &commandLine = commands.front().CommandLine;
  for (size_t index = 1; index < commandLine.size(); ++index) {
    const auto &arg = commandLine[index];
    if (arg == path || arg == commands.front().Filename || arg == "-c") {
      continue;
    }
    if ((arg == "-o" || arg == "/Fo") && index + 1 < commandLine.size()) {
      ++index;
      continue;
    }
    args.push_back(arg);
  }
  if (std::none_of(args.begin(), args.end(),
                   [](const std::string &arg) { return arg.rfind("-std=", 0) == 0; })) {
    args.push_back("-std=c++20");
  }
  return args;
}

llvm::json::Object makeFailure(llvm::StringRef requestId,
                               llvm::StringRef kind,
                               llvm::StringRef message) {
  llvm::json::Object error;
  error["kind"] = kind;
  error["message"] = message;
  error["retryable"] = false;

  llvm::json::Object response;
  response["contract_version"] = kContractVersion;
  response["request_id"] = requestId;
  response["ok"] = false;
  response["error"] = std::move(error);
  return response;
}

int emitJson(llvm::json::Object object) {
  llvm::outs() << llvm::formatv("{0}\n", llvm::json::Value(std::move(object)));
  return 0;
}

}  // namespace

int main() {
  std::string input((std::istreambuf_iterator<char>(std::cin)),
                    std::istreambuf_iterator<char>());

  auto parsed = llvm::json::parse(input);
  if (!parsed) {
    llvm::errs() << "Invalid JSON request.\n";
    return emitJson(makeFailure("", "protocol_error", "Invalid JSON request."));
  }

  auto *request = parsed->getAsObject();
  if (request == nullptr) {
    return emitJson(makeFailure("", "protocol_error", "Request must be a JSON object."));
  }

  const auto requestIdValue = request->getString("request_id");
  const std::string requestId =
      requestIdValue ? requestIdValue->str() : std::string();

  const auto contract = request->getString("contract_version");
  const auto operation = request->getString("operation");
  const auto language = request->getString("language");
  const auto sourceValue = request->getString("source");

  if (!contract || *contract != kContractVersion) {
    return emitJson(makeFailure(
        requestId, "protocol_error", "Unsupported parser backend contract version."));
  }
  if (!operation || *operation != "parse") {
    return emitJson(makeFailure(requestId, "protocol_error", "Unsupported operation."));
  }
  if (!language || *language != "cpp") {
    return emitJson(makeFailure(requestId, "protocol_error", "Expected language cpp."));
  }
  if (!sourceValue) {
    return emitJson(makeFailure(requestId, "protocol_error", "Request source is required."));
  }

  const std::string source = sourceValue->str();
  std::string path = "input.cpp";
  if (const auto pathValue = request->getString("path")) {
    if (!pathValue->empty()) {
      path = pathValue->str();
    }
  }

  std::vector<Entity> entities;
  std::vector<Diagnostic> diagnostics;
  std::vector<std::string> imports;
  collectTextualPreprocessorFacts(source, imports, entities);

  const auto args = compileArgsForPath(path, diagnostics);
  auto ast = clang::tooling::buildASTFromCodeWithArgs(source, args, path, "kadoka-cpp-backend");
  if (!ast) {
    return emitJson(makeFailure(
        requestId, "failure", "Clang could not build even a partial AST."));
  }

  for (auto it = ast->stored_diag_begin(); it != ast->stored_diag_end(); ++it) {
    Diagnostic diagnostic;
    diagnostic.kind = it->getLevel() >= clang::DiagnosticsEngine::Error
                          ? "parse_error"
                          : "clang_diagnostic";
    diagnostic.message = it->getMessage().str();
    if (it->getLocation().isValid()) {
      auto presumed = ast->getSourceManager().getPresumedLoc(it->getLocation());
      if (presumed.isValid()) {
        diagnostic.line = static_cast<int>(presumed.getLine());
      }
    }
    diagnostics.push_back(std::move(diagnostic));
  }

  Visitor visitor(ast->getASTContext(), entities, diagnostics);
  visitor.TraverseDecl(ast->getASTContext().getTranslationUnitDecl());

  llvm::json::Array entityArray;
  for (const auto &entity : entities) {
    entityArray.emplace_back(entityToJson(entity));
  }

  llvm::json::Array diagnosticArray;
  for (const auto &diagnostic : diagnostics) {
    diagnosticArray.emplace_back(diagnosticToJson(diagnostic));
  }

  llvm::json::Object ir;
  ir["language"] = "cpp";
  ir["imports"] = stringsToJson(imports);
  ir["entities"] = std::move(entityArray);
  ir["diagnostics"] = std::move(diagnosticArray);

  llvm::json::Object response;
  response["contract_version"] = kContractVersion;
  response["request_id"] = requestId;
  response["ok"] = true;
  response["backend_id"] = kBackendId;
  response["ir"] = std::move(ir);
  return emitJson(std::move(response));
}
