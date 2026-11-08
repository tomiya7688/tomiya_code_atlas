const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

// {
//   責務: [createGuiHarness: DOMとWails APIをfake化し、実際のGUIイベントを隔離して検証できるようにする,],
// }
function createGuiHarness() {
  const sourceButton = createElement();
  const outputButton = createElement();
  const analyzeButton = createElement();
  const sourcePath = createElement();
  const outputPath = createElement();
  const result = createElement();
  const inputKinds = [createElement(), createElement()];
  inputKinds[0].value = "file";
  inputKinds[1].value = "folder";
  let selectedInputKind = inputKinds[0];
  const htmlPath = path.resolve(__dirname, "../go/internal/ui/web/index.html");
  const operationInputs = [
    ...fs
      .readFileSync(htmlPath, "utf8")
      .matchAll(/type="checkbox" value="([^"]+)"/g),
  ].map((match) => Object.assign(createElement(), { value: match[1] }));
  const documentListeners = new Map();
  const runtimeEvents = new Map();
  const apiCalls = {
    source: 0,
    folder: 0,
    output: 0,
    compatible: [],
    analyze: [],
    smokeResults: [],
  };
  const pendingCompatibility = [];
  let resolveAnalysis;

  const api = {
    sourceFileResult: "E:/project/sample.py",
    projectFolderResult: "E:/project",
    outputFolderResult: "E:/output",
    async OpenSourceFile() {
      apiCalls.source += 1;
      return api.sourceFileResult;
    },
    async OpenProjectFolder() {
      apiCalls.folder += 1;
      return api.projectFolderResult;
    },
    async ChooseOutputFolder() {
      apiCalls.output += 1;
      return api.outputFolderResult;
    },
    async GetCompatibleOperations(source) {
      apiCalls.compatible.push(source);
      if (api.compatibilityError) throw api.compatibilityError;
      if (api.deferCompatibility) {
        return new Promise((resolve, reject) =>
          pendingCompatibility.push({ path: source, resolve, reject }),
        );
      }
      const all = operationInputs.map((input) => input.value);
      const sourceOperations = [
        "call-graph",
        "class-diagram",
        "sequence-diagram",
        "communication-diagram",
        "responsibility-table",
        "comments",
        "static-spec",
        "design-quality",
      ];
      const pythonOperations = [
        "object-diagram",
        "state-diagram",
        "timing-chart",
        "use-case-diagram",
      ];
      const normalizedSource = source.replaceAll("\\", "/");
      if (/\.ya?ml$/i.test(normalizedSource)) {
        return /\.github\/workflows\//i.test(normalizedSource)
          ? ["ci-graph"]
          : [];
      }
      if (normalizedSource === "E:/project") return all;
      if (!/\.(cs|cpp|gd|java|go|py)$/i.test(normalizedSource)) return [];
      return /\.py$/i.test(normalizedSource)
        ? [...sourceOperations, ...pythonOperations]
        : sourceOperations;
    },
    Analyze(source, selectedOperations, output) {
      apiCalls.analyze.push([source, selectedOperations, output]);
      if (api.analysisError) return Promise.reject(api.analysisError);
      return new Promise((resolve) => {
        resolveAnalysis = resolve;
      });
    },
    async ReportSmokeResult(errorMessage) {
      apiCalls.smokeResults.push(errorMessage);
    },
  };

  const elements = new Map([
    ["#source-path", sourcePath],
    ["#output-path", outputPath],
    ["#result", result],
    ["#run-analysis", analyzeButton],
    ["#choose-source", sourceButton],
    ["#choose-output", outputButton],
    ["#operation-compatibility", createElement()],
  ]);
  const document = {
    querySelector(selector) {
      if (selector === 'input[name="input-kind"]:checked')
        return selectedInputKind;
      return elements.get(selector);
    },
    querySelectorAll(selector) {
      if (selector === 'input[name="input-kind"]') return inputKinds;
      if (selector === ".output-option input") return operationInputs;
      if (selector === ".output-option input:checked")
        return operationInputs.filter((input) => input.checked);
      return [];
    },
    addEventListener(name, handler) {
      documentListeners.set(name, handler);
    },
    createElement() {
      return createElement();
    },
  };

  const appPath = path.resolve(__dirname, "../go/internal/ui/web/app.js");
  vm.runInNewContext(fs.readFileSync(appPath, "utf8"), {
    document,
    window: {
      go: { ui: { Application: api } },
      runtime: {
        EventsOn(name, handler) {
          runtimeEvents.set(name, handler);
        },
      },
    },
  });

  return {
    apiCalls,
    api,
    analyzeButton,
    inputKinds,
    operations: operationInputs,
    pendingCompatibility,
    outputButton,
    outputPath,
    resolveAnalysis(value) {
      resolveAnalysis(value);
    },
    emitRuntimeEvent(name, payload) {
      return runtimeEvents.get(name)?.(payload);
    },
    result,
    sourceButton,
    sourcePath,
    selectInputKind(value) {
      selectedInputKind = inputKinds.find((input) => input.value === value);
      selectedInputKind.dispatch("change", { target: selectedInputKind });
    },
    selectOperations(...values) {
      for (const input of operationInputs)
        input.checked = values.includes(input.value);
    },
    keydown(key, modifiers = {}) {
      let prevented = false;
      documentListeners.get("keydown")({
        key,
        altKey: modifiers.altKey ?? true,
        ctrlKey: modifiers.ctrlKey ?? false,
        metaKey: modifiers.metaKey ?? false,
        preventDefault() {
          prevented = true;
        },
      });
      return prevented;
    },
  };
}

// {
//   責務: [createElement: DOM要素の表示状態とevent listenerをNode test runner上で再現する,],
// }
function createElement() {
  const listeners = new Map();
  return {
    children: [],
    className: "",
    classList: { toggle() {} },
    disabled: false,
    checked: false,
    innerHTML: "",
    textContent: "",
    value: "",
    addEventListener(name, handler) {
      listeners.set(name, handler);
    },
    append(child) {
      this.children.push(child);
    },
    click() {
      return listeners.get("click")?.({});
    },
    dispatch(name, event) {
      return listeners.get(name)?.(event);
    },
    replaceChildren(...children) {
      this.children = children;
    },
  };
}

test("rejects missing source, outputs, or output folder before calling the backend", async () => {
  const gui = createGuiHarness();

  await gui.analyzeButton.click();
  assert.match(gui.result.children[0].textContent, /解析するファイル/);

  await gui.sourceButton.click();
  await gui.analyzeButton.click();
  assert.match(gui.result.children[0].textContent, /作りたい成果物/);

  gui.selectOperations("call-graph");
  await gui.analyzeButton.click();
  assert.match(gui.result.children[0].textContent, /保存先フォルダー/);
  assert.equal(gui.apiCalls.analyze.length, 0);
});

test("submits all supported operations once and accepts a missing output path list", async () => {
  const gui = createGuiHarness();
  gui.selectInputKind("folder");
  await gui.sourceButton.click();
  await gui.outputButton.click();
  const supportedOperations = [
    "call-graph",
    "class-diagram",
    "sequence-diagram",
    "object-diagram",
    "communication-diagram",
    "state-diagram",
    "timing-chart",
    "use-case-diagram",
    "responsibility-table",
    "package-diagram",
    "component-diagram",
    "deployment-diagram",
    "ci-graph",
    "comments",
    "static-spec",
    "design-quality",
  ];
  const htmlPath = path.resolve(__dirname, "../go/internal/ui/web/index.html");
  const html = fs.readFileSync(htmlPath, "utf8");
  const renderedOperations = [
    ...html.matchAll(/type="checkbox" value="([^"]+)"/g),
  ].map((match) => match[1]);
  assert.deepEqual(renderedOperations, supportedOperations);
  gui.selectOperations(...supportedOperations);

  const analysis = gui.analyzeButton.click();
  assert.equal(gui.analyzeButton.disabled, true);
  await gui.analyzeButton.click();
  assert.equal(gui.apiCalls.analyze.length, 1);
  assert.equal(
    JSON.stringify(gui.apiCalls.analyze[0]),
    JSON.stringify(["E:/project", supportedOperations, "E:/output"]),
  );
  assert.equal(gui.keydown("r"), true);
  assert.equal(gui.apiCalls.analyze.length, 1);

  gui.resolveAnalysis({ message: "解析が完了しました。", outputPaths: null });
  await analysis;
  assert.equal(gui.analyzeButton.disabled, false);
  assert.equal(gui.result.children[0].textContent, "解析が完了しました。");
  assert.equal(gui.result.children[1].children.length, 0);
});

test("enables operations that match the selected input kind", async () => {
  const gui = createGuiHarness();
  const operation = (value) =>
    gui.operations.find((input) => input.value === value);
  const enabledCount = () =>
    gui.operations.filter((input) => !input.disabled).length;

  assert.equal(operation("package-diagram").disabled, true);
  assert.equal(operation("ci-graph").disabled, true);
  await gui.sourceButton.click();
  assert.equal(operation("call-graph").disabled, false);
  assert.equal(operation("package-diagram").disabled, true);
  assert.equal(enabledCount(), 12);

  gui.api.sourceFileResult = "E:/project/.github/workflows/ci.yaml";
  await gui.sourceButton.click();
  assert.equal(operation("ci-graph").disabled, false);
  assert.equal(operation("call-graph").disabled, true);
  assert.equal(enabledCount(), 1);
  gui.selectOperations("ci-graph");

  gui.api.sourceFileResult = "E:/project/config/settings.yaml";
  await gui.sourceButton.click();
  assert.equal(enabledCount(), 0);

  gui.api.sourceFileResult = "E:/project/SAMPLE.PY";
  await gui.sourceButton.click();
  assert.equal(operation("ci-graph").disabled, true);
  assert.equal(operation("ci-graph").checked, false);
  assert.equal(enabledCount(), 12);

  gui.api.sourceFileResult = "E:/project/App.cs";
  await gui.sourceButton.click();
  assert.equal(operation("object-diagram").disabled, true);
  assert.equal(enabledCount(), 8);

  gui.selectInputKind("folder");
  await gui.sourceButton.click();
  assert.equal(operation("package-diagram").disabled, false);
  assert.equal(operation("deployment-diagram").disabled, false);
  assert.equal(operation("ci-graph").disabled, false);
  assert.equal(enabledCount(), 16);
});

test("keeps the newest compatibility response when selections overlap", async () => {
  const gui = createGuiHarness();
  const operation = (value) =>
    gui.operations.find((input) => input.value === value);
  gui.api.deferCompatibility = true;

  gui.api.sourceFileResult = "E:/project/first.go";
  const firstSelection = gui.sourceButton.click();
  await new Promise((resolve) => setImmediate(resolve));
  gui.api.sourceFileResult = "E:/project/second.go";
  const secondSelection = gui.sourceButton.click();
  await new Promise((resolve) => setImmediate(resolve));

  gui.pendingCompatibility[1].resolve(["class-diagram"]);
  await secondSelection;
  gui.pendingCompatibility[0].resolve(["call-graph"]);
  await firstSelection;

  assert.equal(operation("class-diagram").disabled, false);
  assert.equal(operation("call-graph").disabled, true);
});

test("ignores compatibility responses after changing input kind", async () => {
  const gui = createGuiHarness();
  const operation = (value) =>
    gui.operations.find((input) => input.value === value);
  gui.api.deferCompatibility = true;

  const selection = gui.sourceButton.click();
  await new Promise((resolve) => setImmediate(resolve));
  gui.selectInputKind("folder");
  gui.pendingCompatibility[0].resolve(["call-graph"]);
  await selection;

  assert.equal(operation("call-graph").disabled, true);
  assert.equal(gui.sourcePath.textContent, "まだ選択されていません");
});

test("selects project folders and preserves paths when dialogs are cancelled", async () => {
  const gui = createGuiHarness();

  await gui.sourceButton.click();
  assert.equal(gui.sourcePath.textContent, "E:/project/sample.py");
  gui.selectInputKind("folder");
  assert.equal(gui.sourcePath.textContent, "まだ選択されていません");
  await gui.sourceButton.click();
  assert.equal(gui.apiCalls.source, 1);
  assert.equal(gui.apiCalls.folder, 1);
  assert.equal(gui.sourcePath.textContent, "E:/project");

  gui.api.projectFolderResult = "";
  await gui.sourceButton.click();
  assert.equal(gui.sourcePath.textContent, "E:/project");

  await gui.outputButton.click();
  assert.equal(gui.outputPath.textContent, "E:/output");
  gui.api.outputFolderResult = "";
  await gui.outputButton.click();
  assert.equal(gui.outputPath.textContent, "E:/output");
});

// {
//   責務: [検証: Alt+O/P/Rを各GUI操作へ結び付け、修飾キー付きの誤操作を無視する,]
// }
test("routes supported Alt shortcuts to their GUI actions", async () => {
  const gui = createGuiHarness();

  assert.equal(gui.keydown("x"), false);
  assert.equal(gui.keydown("o", { ctrlKey: true }), false);
  assert.equal(gui.apiCalls.source, 0);
  assert.equal(gui.keydown("o"), true);
  assert.equal(gui.apiCalls.source, 1);
  assert.equal(gui.keydown("p"), true);
  assert.equal(gui.apiCalls.output, 1);

  assert.equal(gui.keydown("r", { altKey: false }), false);
  assert.equal(gui.keydown("r", { metaKey: true }), false);
  assert.equal(gui.apiCalls.analyze.length, 0);

  await gui.sourceButton.click();
  gui.selectOperations("call-graph");
  await gui.outputButton.click();
  assert.equal(gui.keydown("r"), true);
  assert.equal(gui.apiCalls.analyze.length, 1);
  assert.equal(
    JSON.stringify(gui.apiCalls.analyze[0]),
    JSON.stringify(["E:/project/sample.py", ["call-graph"], "E:/output"]),
  );
  gui.resolveAnalysis({ message: "解析が完了しました。", outputPaths: [] });
  await new Promise((resolve) => setImmediate(resolve));
});

// {
//   責務: [検証: 配布smoke eventが入力の互換性照会結果12件を一括選択して解析bindingを通ることを確認する,]
// }
test("runs packaged GUI smoke through frontend compatibility and Analyze flow", async () => {
  const gui = createGuiHarness();
  const smokeRun = gui.emitRuntimeEvent("gui-smoke-analyze", {
    sourcePath: "E:/project/sample.py",
    operations: [],
    allCompatibleOperations: true,
    outputDirectory: "E:/gui-output",
  });

  await new Promise((resolve) => setImmediate(resolve));
  assert.deepEqual(gui.apiCalls.compatible, ["E:/project/sample.py"]);
  assert.equal(gui.apiCalls.analyze.length, 1);
  assert.equal(gui.apiCalls.analyze[0][1].length, 12);
  assert.equal(
    JSON.stringify(gui.apiCalls.analyze[0][1].slice().sort()),
    JSON.stringify(
      [
        "call-graph",
        "class-diagram",
        "sequence-diagram",
        "object-diagram",
        "communication-diagram",
        "state-diagram",
        "timing-chart",
        "use-case-diagram",
        "responsibility-table",
        "comments",
        "static-spec",
        "design-quality",
      ].sort(),
    ),
  );
  for (const operation of gui.apiCalls.analyze[0][1]) {
    assert.equal(
      gui.operations.find((input) => input.value === operation).checked,
      true,
      `${operation} must be selected in the GUI`,
    );
  }

  gui.resolveAnalysis({
    message: "解析が完了しました。12件の成果物を保存しました。",
    outputPaths: [],
  });
  await smokeRun;
  assert.deepEqual(gui.apiCalls.smokeResults, [""]);
});

// {
//   責務: [検証: 入力に非対応なGUI smoke成果物を拒否してGoへ診断を返すことを確認する,]
// }
test("reports incompatible GUI smoke operations without invoking analysis", async () => {
  const gui = createGuiHarness();
  await gui.emitRuntimeEvent("gui-smoke-analyze", {
    sourcePath: "E:/project/Unsupported.cs",
    operations: ["object-diagram"],
    outputDirectory: "E:/gui-output",
  });

  assert.equal(gui.apiCalls.analyze.length, 0);
  assert.match(gui.apiCalls.smokeResults[0], /成果物を選べません/);
});

// {
//   責務: [検証: GUI smoke要求・互換性確認・解析bindingの失敗理由をGoへ通知する,]
// }
test("reports GUI smoke failures without hiding the failing stage", async (t) => {
  const scenarios = [
    {
      name: "malformed request",
      request: {
        sourcePath: "E:/project/sample.py",
        operations: [],
        outputDirectory: "E:/gui-output",
      },
      expectedMessage: /成果物が選ばれていません/,
      expectedAnalysisCalls: 0,
    },
    {
      name: "compatibility lookup",
      request: {
        sourcePath: "E:/project/sample.py",
        operations: ["call-graph"],
        outputDirectory: "E:/gui-output",
      },
      configure(gui) {
        gui.api.compatibilityError = new Error("入力の種類を確認できません。");
      },
      expectedMessage: /成果物を調べられません/,
      expectedAnalysisCalls: 0,
    },
    {
      name: "analysis binding",
      request: {
        sourcePath: "E:/project/sample.py",
        operations: ["call-graph"],
        outputDirectory: "E:/gui-output",
      },
      configure(gui) {
        gui.api.analysisError = new Error("解析を完了できません。");
      },
      expectedMessage: /解析を完了できません/,
      expectedAnalysisCalls: 1,
    },
  ];

  for (const scenario of scenarios) {
    await t.test(scenario.name, async () => {
      const gui = createGuiHarness();
      scenario.configure?.(gui);
      await gui.emitRuntimeEvent("gui-smoke-analyze", scenario.request);

      assert.equal(gui.apiCalls.analyze.length, scenario.expectedAnalysisCalls);
      assert.match(gui.apiCalls.smokeResults[0], scenario.expectedMessage);
    });
  }
});
