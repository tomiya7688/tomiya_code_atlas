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
    if (requestId !== compatibilityRequestId) return;
    compatibleOperations = new Set(operations);
  } catch (error) {
    if (requestId !== compatibilityRequestId) return;
    compatibleOperations = new Set();
    showMessage(String(error), "error");
  }
  updateOperationCompatibility();
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

analyzeButton.addEventListener("click", async () => {
  if (analyzeButton.disabled) return;

  const operations = [
    ...document.querySelectorAll(".output-option input:checked"),
  ].map((item) => item.value);
  if (!selected.source) {
    showMessage(
      "解析するファイルまたはプロジェクトフォルダーを選んでください。",
      "error",
    );
    return;
  }
  if (operations.length === 0) {
    showMessage("作りたい成果物を一つ以上選んでください。", "error");
    return;
  }
  if (!selected.output) {
    showMessage("保存先フォルダーを選んでください。", "error");
    return;
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
  } catch (error) {
    showMessage(String(error).replace(/^Error:\s*/, ""), "error");
  } finally {
    analyzeButton.disabled = false;
  }
});

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
