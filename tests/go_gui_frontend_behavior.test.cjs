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
  const operations = [];
  const documentListeners = new Map();
  const apiCalls = { source: 0, folder: 0, output: 0, analyze: [] };
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
    Analyze(source, selectedOperations, output) {
      apiCalls.analyze.push([source, selectedOperations, output]);
      return new Promise((resolve) => { resolveAnalysis = resolve; });
    },
  };

  const elements = new Map([
    ["#source-path", sourcePath],
    ["#output-path", outputPath],
    ["#result", result],
    ["#run-analysis", analyzeButton],
    ["#choose-source", sourceButton],
    ["#choose-output", outputButton],
  ]);
  const document = {
    querySelector(selector) {
      if (selector === 'input[name="input-kind"]:checked') return selectedInputKind;
      return elements.get(selector);
    },
    querySelectorAll(selector) {
      if (selector === 'input[name="input-kind"]') return inputKinds;
      if (selector === ".diagram input:checked") return operations;
      return [];
    },
    addEventListener(name, handler) { documentListeners.set(name, handler); },
    createElement() { return createElement(); },
  };

  const appPath = path.resolve(__dirname, "../go/internal/ui/web/app.js");
  vm.runInNewContext(fs.readFileSync(appPath, "utf8"), {
    document,
    window: { go: { ui: { Application: api } } },
  });

  return {
    apiCalls,
    api,
    analyzeButton,
    inputKinds,
    operations,
    outputButton,
    outputPath,
    resolveAnalysis(value) { resolveAnalysis(value); },
    result,
    sourceButton,
    sourcePath,
    selectInputKind(value) {
      selectedInputKind = inputKinds.find((input) => input.value === value);
      selectedInputKind.dispatch("change", { target: selectedInputKind });
    },
    keydown(key, modifiers = {}) {
      let prevented = false;
      documentListeners.get("keydown")({
        key,
        altKey: modifiers.altKey ?? true,
        ctrlKey: modifiers.ctrlKey ?? false,
        metaKey: modifiers.metaKey ?? false,
        preventDefault() { prevented = true; },
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
    innerHTML: "",
    textContent: "",
    value: "",
    addEventListener(name, handler) { listeners.set(name, handler); },
    append(child) { this.children.push(child); },
    click() { return listeners.get("click")?.({}); },
    dispatch(name, event) { return listeners.get(name)?.(event); },
    replaceChildren(...children) { this.children = children; },
  };
}

test("rejects missing source, diagrams, or output before calling the backend", async () => {
  const gui = createGuiHarness();

  await gui.analyzeButton.click();
  assert.match(gui.result.children[0].textContent, /解析するファイル/);

  await gui.sourceButton.click();
  await gui.analyzeButton.click();
  assert.match(gui.result.children[0].textContent, /作りたい図/);

  gui.operations.push({ value: "call-graph" });
  await gui.analyzeButton.click();
  assert.match(gui.result.children[0].textContent, /保存先フォルダー/);
  assert.equal(gui.apiCalls.analyze.length, 0);
});

test("submits multiple diagrams once and accepts a missing output path list", async () => {
  const gui = createGuiHarness();
  await gui.sourceButton.click();
  await gui.outputButton.click();
  gui.operations.push({ value: "call-graph" }, { value: "class-diagram" });

  const analysis = gui.analyzeButton.click();
  assert.equal(gui.analyzeButton.disabled, true);
  await gui.analyzeButton.click();
  assert.equal(gui.apiCalls.analyze.length, 1);
  assert.equal(JSON.stringify(gui.apiCalls.analyze[0]), JSON.stringify([
    "E:/project/sample.py",
    ["call-graph", "class-diagram"],
    "E:/output",
  ]));
  assert.equal(gui.keydown("r"), true);
  assert.equal(gui.apiCalls.analyze.length, 1);

  gui.resolveAnalysis({ message: "解析が完了しました。", outputPaths: null });
  await analysis;
  assert.equal(gui.analyzeButton.disabled, false);
  assert.equal(gui.result.children[0].textContent, "解析が完了しました。");
  assert.equal(gui.result.children[1].children.length, 0);
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

test("prevents browser defaults only for supported Alt shortcuts", async () => {
  const gui = createGuiHarness();

  assert.equal(gui.keydown("x"), false);
  assert.equal(gui.keydown("o", { ctrlKey: true }), false);
  assert.equal(gui.apiCalls.source, 0);
  assert.equal(gui.keydown("o"), true);
  assert.equal(gui.apiCalls.source, 1);
  assert.equal(gui.keydown("p"), true);
  assert.equal(gui.apiCalls.output, 1);
});
