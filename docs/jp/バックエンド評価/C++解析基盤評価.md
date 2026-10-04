# C++ AST／意味解析バックエンドの評価

Issue #135ではC++解析backendを評価し、Issue #191では実文法parserとhelper配布を実装します。

## 選定

C++のprimary parserとして、**Clangのlibclang translation-unit parser**を選びます。`libclang-ng`に含まれるClang 22.1.4.2のcompiler parserをPython one-dir helperへ同梱し、利用者にLLVM／Clangの導入を要求しません。

```text
backend id: cpp-libclang-helper
hosting: libclang shared libraryとCPythonを含む同梱one-dir helper
protocol: Parser Backend Contract v1 JSON
```

libclangはClang compilerのgrammar parserとtranslation-unit ASTを実行します。helperはnamespace、class／struct、template parameter、base class、function、method、field、include、call、source locationとcompiler diagnosticをCommon IRのsubsetへ写します。`compile_commands.json`があれば定義、include path、言語versionなどの引数を使います。

libclang C APIが公開するAST／Sema情報はLibToolingより限定されています。このIssueで必要な構造parserを実装し、将来template instantiationやより深いoverload/Sema解析を必要とする機能ではLibToolingへ移行します。Common IRへ現在写さないC++ compiler内部情報を、実装済みと扱いません。

tree-sitter-cppは、未完成のeditor bufferを扱うための構文fallback候補として残します。

## Clang toolingを選ぶ理由

C++の意味解析は次の要素に大きく左右されます。

- preprocessor macro
- include path
- compile definition／flag
- overload解決
- templateとinstantiation
- 継承
- 言語標準／version
- platform／toolchain header

compiler parserは実際のtranslation unitを構築するため、compile databaseのflagを再利用できます。compile databaseやvendor SDK headerがない場合でもソース文法をparseし、既知のnamespace/class/functionを出力して不足をdiagnosticとして残します。

## Fixtureの追加要素

C++ fixtureには次を含めます。

- system／project include
- macro展開
- generic template class
- 継承
- overloaded method
- 同じ関数の複数回呼び出し
- async／future使用
- lambda／nested scope

これは能力プローブです。ClangのDecl／Stmt kind、SourceManager objectなどcompiler固有の型は、Language Adapter／backendの境界より外へ出しません。

## 候補比較

| 候補 | template／overload | preprocessor／include | 不完全なソース | 配布 | 判断 |
| --- | --- | --- | --- | --- | --- |
| libclang compiler parser / C API | AST subsetと解決済み参照 | compile database経由 | 診断と回復可能AST | libclang + CPython one-dir | **初期parserとして選定** |
| Clang LibTooling | 全C++ AST／Semaへ最も広くアクセス | compiler tooling全体 | 診断と部分復旧 | 大型native helper + Clang assets | 高度なsemantic follow-up |
| tree-sitter-cpp | 構文のみ | 構文上のpreprocessor node | 強い | native parser assets | fallback |
| 現行regex adapter | コメント向けheuristicのみ | 意味モデルなし | 該当なし | 依存が軽い | 解析backendではない |

## Project設定

`compile_commands.json` があれば優先します。translation unitで実際に使われるflagを把握する最善の情報源です。

compile databaseがない場合、Tomiyaは保守的な既定値と検出済みinclude pathを利用できますが、結果を部分的なものとして明示します。header欠落や未解決symbolは、解決済みであるかのように作り出さず未解決のまま残します。

## 配布

Windowsアプリの実行に外部LLVM／Clangのインストールを要求しません。

Windows配布には、libclang、CPython runtime、helper licenseを同梱します。SDK headerのない環境でgrammar parseが継続し、missing headerをdiagnosticにします。プロジェクト固有のsemantic completenessには、引き続き実プロジェクトのcompile databaseとSDK headerが必要です。

想定配置:

```text
backends/
  cpp/
    tomiya-cpp-backend.exe
    _internal/clang/native/libclang.dll
    licenses/LLVM-LICENSE.txt
```

Clang本体を同梱しても、system／vendor headerへの依存は解消しません。たとえばMSVC標準libraryの正確な意味解析には、解析対象projectが使うtoolchain／SDKのheaderが必要な場合があります。設定が欠けても可能な範囲の部分的な構造結果を返し、不足を報告します。

## ライセンス

LLVM／Clangは **Apache-2.0 WITH LLVM-exception** です。tree-sitter-cppはMITです。

## 今後の作業

初期helperはcompiler grammarのfull parseとCommon IR structural subsetを提供します。LibToolingへの移行は、より深いsemantic factsが必要になった時点で別Issueにします。

2026-09-18時点で確認した情報源:

- https://github.com/llvm/llvm-project
- https://github.com/llvm/llvm-project/blob/main/clang/include/clang-c/Index.h
- https://github.com/tree-sitter/tree-sitter-cpp
