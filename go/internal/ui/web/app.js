const selected = { source: "", output: "" };
let compatibleOperations = new Set();
let compatibilityRequestId = 0;
const sourcePath = document.querySelector("#source-path");
const outputPath = document.querySelector("#output-path");
const result = document.querySelector("#result");
const analyzeButton = document.querySelector("#run-analysis");
const operationCompatibility = document.querySelector(
  "#operation-compatibility",
);
function updateOperationCompatibility() {
  for (const input of document.querySelectorAll(".output-option input")) {
    input.disabled = !compatibleOperations.has(input.value);
    if (input.disabled) input.checked = false;
  }
  operationCompatibility.textContent = compatibleOperations.size
    ? `この入力で実行できる成果物を有効にしました（${compatibleOperations.size}種類）。`
    : "対応するソースまたはGitHub Actions workflowを選んでください。";
}

async function loadCompatibleOperations(path) {
  const requestId = ++compatibilityRequestId;
  try {
    const operations =
      await window.go.ui.Application.GetCompatibleOperations(path);
    if (requestId !== compatibilityRequestId) return false;
    compatibleOperations = new Set(operations);
  } catch (error) {
    if (requestId !== compatibilityRequestId) return false;
    compatibleOperations = new Set();
    showMessage(String(error), "error");
    updateOperationCompatibility();
    return false;
  }
  updateOperationCompatibility();
  return true;
}

function showPath(element, path) {
  element.textContent = path || "まだ選択されていません";
  element.classList.toggle("empty", !path);
}

function showMessage(message, kind = "") {
  result.className = `result ${kind}`;
  result.replaceChildren(
    Object.assign(document.createElement("p"), { textContent: message }),
  );
}

document.querySelector("#choose-source").addEventListener("click", async () => {
  try {
    const kind = document.querySelector(
      'input[name="input-kind"]:checked',
    ).value;
    const path =
      kind === "file"
        ? await window.go.ui.Application.OpenSourceFile()
        : await window.go.ui.Application.OpenProjectFolder();
    if (path) {
      selected.source = path;
      showPath(sourcePath, path);
      await loadCompatibleOperations(path);
    }
  } catch (error) {
    showMessage(String(error), "error");
  }
});

document.querySelectorAll('input[name="input-kind"]').forEach((input) =>
  input.addEventListener("change", (event) => {
    compatibilityRequestId += 1;
    document.querySelector("#choose-source").innerHTML =
      `${event.target.value === "file" ? "ファイルを選ぶ" : "フォルダーを選ぶ"} <kbd>Alt+O</kbd>`;
    selected.source = "";
    compatibleOperations = new Set();
    showPath(sourcePath, "");
    updateOperationCompatibility();
  }),
);

document.querySelector("#choose-output").addEventListener("click", async () => {
  try {
    const path = await window.go.ui.Application.ChooseOutputFolder();
    if (path) {
      selected.output = path;
      showPath(outputPath, path);
    }
  } catch (error) {
    showMessage(String(error), "error");
  }
});

// {
//   責務: [analyzeSelection: 選択済みの入力・成果物・保存先を検証し、画面の解析bindingを呼び出す,]
// }
async function analyzeSelection() {
  if (analyzeButton.disabled) return "解析を実行中です。";

  const operations = [
    ...document.querySelectorAll(".output-option input:checked"),
  ].map((item) => item.value);
  if (!selected.source) {
    showMessage(
      "解析するファイルまたはプロジェクトフォルダーを選んでください。",
      "error",
    );
    return "解析するファイルまたはプロジェクトフォルダーを選んでください。";
  }
  if (operations.length === 0) {
    showMessage("作りたい成果物を一つ以上選んでください。", "error");
    return "作りたい成果物を一つ以上選んでください。";
  }
  if (!selected.output) {
    showMessage("保存先フォルダーを選んでください。", "error");
    return "保存先フォルダーを選んでください。";
  }

  analyzeButton.disabled = true;
  showMessage("解析しています。ファイル数によっては時間がかかります。");
  try {
    const response = await window.go.ui.Application.Analyze(
      selected.source,
      operations,
      selected.output,
    );
    result.className = "result success";
    const message = document.createElement("p");
    message.textContent = response.message;
    const paths = document.createElement("ul");
    const outputPaths = Array.isArray(response?.outputPaths)
      ? response.outputPaths
      : [];
    outputPaths.forEach((path) =>
      paths.append(
        Object.assign(document.createElement("li"), { textContent: path }),
      ),
    );
    result.replaceChildren(message, paths);
    return "";
  } catch (error) {
    const errorMessage = String(error).replace(/^Error:\s*/, "");
    showMessage(errorMessage, "error");
    return errorMessage;
  } finally {
    analyzeButton.disabled = false;
  }
}

analyzeButton.addEventListener("click", analyzeSelection);

// {
//   責務: [runGuiSmoke: hidden smoke requestを画面状態へ反映し、互換性のある全成果物を選んで利用者と同じ解析処理を実行する,]
// }
async function runGuiSmoke(request) {
  try {
    const selectAllCompatibleOperations =
      request?.allCompatibleOperations === true;
    if (!request?.sourcePath || !request?.outputDirectory) {
      throw new Error("GUI smoke testの入力fileと保存先を取得できません。");
    }
    if (!Array.isArray(request.operations)) {
      throw new Error("GUI smoke testで成果物が選ばれていません。");
    }
    if (!selectAllCompatibleOperations && request.operations.length === 0) {
      throw new Error("GUI smoke testで成果物が選ばれていません。");
    }

    selected.source = request.sourcePath;
    showPath(sourcePath, selected.source);
    if (!(await loadCompatibleOperations(selected.source))) {
      throw new Error("GUI smoke testの入力に対応する成果物を調べられません。");
    }
    const operationInputs = [
      ...document.querySelectorAll(".output-option input"),
    ];
    const knownOperations = new Set(
      operationInputs.map((input) => input.value),
    );
    // GUI自身の互換性照会を使い、smoke専用の重複operation一覧を持たないようにします。
    const selectedOperations = selectAllCompatibleOperations
      ? [...compatibleOperations]
      : request.operations;
    if (selectedOperations.length === 0) {
      throw new Error("GUI smoke testの入力に対応する成果物がありません。");
    }
    for (const operation of selectedOperations) {
      if (
        !knownOperations.has(operation) ||
        !compatibleOperations.has(operation)
      ) {
        throw new Error(
          `GUI smoke testの入力では成果物を選べません: ${operation}`,
        );
      }
    }
    for (const input of operationInputs) {
      input.checked = selectedOperations.includes(input.value);
    }
    selected.output = request.outputDirectory;
    showPath(outputPath, selected.output);
    const errorMessage = await analyzeSelection();
    await window.go.ui.Application.ReportSmokeResult(errorMessage);
  } catch (error) {
    const errorMessage = String(error).replace(/^Error:\s*/, "");
    showMessage(errorMessage, "error");
    await window.go.ui.Application.ReportSmokeResult(errorMessage);
  }
}

window.runtime.EventsOn("gui-smoke-analyze", runGuiSmoke);

document.addEventListener("keydown", (event) => {
  if (!event.altKey || event.ctrlKey || event.metaKey) return;
  const key = event.key.toLowerCase();
  if (!["o", "p", "r"].includes(key)) return;
  event.preventDefault();
  if (key === "o") document.querySelector("#choose-source").click();
  if (key === "p") document.querySelector("#choose-output").click();
  if (key === "r" && !analyzeButton.disabled) analyzeButton.click();
});

updateOperationCompatibility();
