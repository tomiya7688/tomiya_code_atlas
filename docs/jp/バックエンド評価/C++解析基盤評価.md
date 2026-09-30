# C++ AST／意味解析バックエンドの評価

Issue #135では、C++解析バックエンドを共通適合fixtureに照らして評価します。

## 選定

C++の主バックエンドとして、**Clang LibToolingを使う完全なC++ AST helper**を選びます。

```text
backend id: cpp-clang-tooling-helper
hosting: 同梱native helper
protocol: Parser Backend Contract v1 JSON
```

libclangのC APIは主たる意味解析backendにしません。公開されている `clang-c/Index.h` の説明では、C interfaceは小さく安定して保つことを意図しており、Clang C++ ASTが持つ全情報を公開しないと明記されています。Tomiyaが目指すtemplate、overload、call targetの解析には、より豊富なcompiler AST／Sema層が必要です。

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

構文だけのparserではこれらを確実に再構成できません。Clang helperならtranslation unitを構築してcompilerの意味情報を利用できます。

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
| Clang LibTooling／完全なC++ AST helper | compilerの情報を利用 | compilerの情報を利用 | 診断と部分復旧 | native helper + Clang assets | **選定** |
| libclang C API | 高水準の部分集合 | 利用できるがAPIを意図的に限定 | 診断／AST | libclang DLL | 主backendにはしない |
| tree-sitter-cpp | 構文のみ | 構文上のpreprocessor node | 強い | native parser assets | fallback |
| 現行regex adapter | コメント向けheuristicのみ | 意味モデルなし | 該当なし | 依存が軽い | 解析backendではない |

## Project設定

`compile_commands.json` があれば優先します。translation unitで実際に使われるflagを把握する最善の情報源です。

compile databaseがない場合、Tomiyaは保守的な既定値と検出済みinclude pathを利用できますが、結果を部分的なものとして明示します。header欠落や未解決symbolは、解決済みであるかのように作り出さず未解決のまま残します。

## 配布

Windowsアプリの実行に外部LLVM／Clangのインストールを要求しません。

想定配置:

```text
backends/
  cpp/
    tomiya-cpp-backend.exe
    clang/llvm runtime DLLまたは静的link相当
    lib/clang/<version>/include/...   # Clang組み込みresource header
```

Clang本体を同梱しても、system／vendor headerへの依存は解消しません。たとえばMSVC標準libraryの正確な意味解析には、解析対象projectが使うtoolchain／SDKのheaderが必要な場合があります。設定が欠けても可能な範囲の部分的な構造結果を返し、不足を報告します。

## ライセンス

LLVM／Clangは **Apache-2.0 WITH LLVM-exception** です。tree-sitter-cppはMITです。

## 今後の作業

#135はbackend選定のみを記録します。native helper、compile database対応、Common IR変換、backend manifest、Windows onedir配布を実装するには別Issueが必要です。

2026-09-18時点で確認した情報源:

- https://github.com/llvm/llvm-project
- https://github.com/llvm/llvm-project/blob/main/clang/include/clang-c/Index.h
- https://github.com/tree-sitter/tree-sitter-cpp
