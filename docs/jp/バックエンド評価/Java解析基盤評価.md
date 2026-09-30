# Java AST／意味解析バックエンドの評価

Issue #136では、Java parser／意味解析backendを#131の共通適合fixtureに照らして評価します。

## 選定

Javaの主たる構文／意味解析backendとして、同梱Java helperで動かす **JavaParser + JavaSymbolSolver** を選びます。

```text
backend id: java-javaparser-symbol-solver-helper
role: 主構文 + 意味解析
transport: Parser Backend Contract v1 JSON
runtime: 同梱private JVM
```

Tomiyaの解析を実行するために、利用者へJDK、JRE、Maven、Gradleの導入を要求しません。

## JavaParser + Symbol Solverを選ぶ理由

Tomiyaのclass／package図とcall／関係解析には、Java構文以外に宣言解決、import、継承／interface、generic type、overload、lambdaの情報が役立ちます。

JavaParserは現代Javaの解析と高度な解析機能を提供し、Symbol Solverは宣言と型の解決を目的としています。構文だけのtree-sitterより適合し、Eclipse JDTより独立helperの範囲を小さく保てます。

この評価で確認した最新JavaParser releaseは **3.28.2**（2026-05-31）です。2026年9月時点でrepositoryの更新も続いていました。

## 候補比較

| 候補 | 構文 | symbol／overload | generic／lambda | 配布 | 判断 |
| --- | --- | --- | --- | --- | --- |
| JavaParser + Symbol Solver | 現代Java AST | project／source／JARを考慮した解決 | 強い意味解析 | helper JAR + private JVM | **選定** |
| Eclipse JDT Core | compiler級 | 非常に強いbinding／compiler model | 非常に強い | Eclipse／JDT連携の面が大きい | 主backendにはしない |
| tree-sitter-java | 強いincremental syntax | 単独ではなし | 構文のみ | 小さなnative parser | 未完成buffer用fallback |
| 現行regex comment adapter | 限定的 | なし | なし | 依存が軽い | コメント生成専用 |

## 配布

実装時には次を同梱します。

```text
backends/java/
├─ tomiya-java-backend.jar
└─ runtime/
   └─ bin/java.exe
```

build中にredistributableなprivate OpenJDK runtimeを組み立てます。依存を確認した後に最小化した `jlink` runtimeを使えますが、trimmingによってSymbol Solverが気付かず壊れることがないようにします。

Python hostは定義済みParser Backend Contract v1を介してhelperを起動します。stdoutはprotocol専用、diagnosticはstderrです。

## Classpath／project意味解析

利用可能な場合、helperは次を解決します。

- project source root
- importとpackage
- interfaceと継承
- overloaded method
- generic type parameterとargument
- lambda／functional interfaceのtarget
- 依存JAR／classpath entry

Maven／Gradle metadataは依存の発見に役立つ場合がありますが、基本的なJava sourceを解析するだけで利用者にMaven／Gradleを導入させません。外部依存が不足する場合はsymbolを作り出さず、未解決の事実として残します。

## ライセンス

JavaParserはLGPLまたはApache License 2.0で提供されます。Tomiyaでは **Apache-2.0** を利用し、必要なnoticeを保持します。

Eclipse JDT CoreはEPL-2.0、tree-sitter-javaはMITです。

## 適合fixture

Java fixtureには次を含めます。

- `package`
- 複数の `import`
- interface実装
- class継承
- generic class／interface
- overloaded helper method
- method callの複数回出現
- lambda
- CompletionStage／CompletableFutureを使う非同期形式のflow

JavaParser／JDT／tree-sitterのnode名はCommon IR goldenに加えません。

## 今後の作業

#136は選定のみです。Java helper、Symbol Solver連携、Common IRへの正規化、private JVMのWindows onedir同梱、#131 fixtureによるCI検証には別の実装Issueが必要です。

2026-09-18時点で確認した情報源:

- https://github.com/javaparser/javaparser
- https://github.com/eclipse-jdt/eclipse.jdt.core
- https://github.com/tree-sitter/tree-sitter-java
