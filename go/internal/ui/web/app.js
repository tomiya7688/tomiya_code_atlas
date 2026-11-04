const selected = { source: "", output: "" };
const sourcePath = document.querySelector("#source-path");
const outputPath = document.querySelector("#output-path");
const result = document.querySelector("#result");
const analyzeButton = document.querySelector("#run-analysis");

function showPath(element, path) {
  element.textContent = path || "まだ選択されていません";
  element.classList.toggle("empty", !path);
}

function showMessage(message, kind = "") {
  result.className = `result ${kind}`;
  result.replaceChildren(Object.assign(document.createElement("p"), { textContent: message }));
}

document.querySelector("#choose-source").addEventListener("click", async () => {
  try {
    const kind = document.querySelector('input[name="input-kind"]:checked').value;
    const path = kind === "file"
      ? await window.go.ui.Application.OpenSourceFile()
      : await window.go.ui.Application.OpenProjectFolder();
    if (path) { selected.source = path; showPath(sourcePath, path); }
  } catch (error) { showMessage(String(error), "error"); }
});

document.querySelectorAll('input[name="input-kind"]').forEach((input) => input.addEventListener("change", (event) => {
  document.querySelector("#choose-source").innerHTML = `${event.target.value === "file" ? "ファイルを選ぶ" : "フォルダーを選ぶ"} <kbd>Alt+O</kbd>`;
  selected.source = "";
  showPath(sourcePath, "");
}));

document.querySelector("#choose-output").addEventListener("click", async () => {
  try {
    const path = await window.go.ui.Application.ChooseOutputFolder();
    if (path) { selected.output = path; showPath(outputPath, path); }
  } catch (error) { showMessage(String(error), "error"); }
});

analyzeButton.addEventListener("click", async () => {
  const operations = [...document.querySelectorAll(".diagram input:checked")].map((item) => item.value);
  analyzeButton.disabled = true;
  showMessage("解析しています。ファイル数によっては時間がかかります。");
  try {
    const response = await window.go.ui.Application.Analyze(selected.source, operations, selected.output);
    result.className = "result success";
    const message = document.createElement("p");
    message.textContent = response.message;
    const paths = document.createElement("ul");
    response.outputPaths.forEach((path) => paths.append(Object.assign(document.createElement("li"), { textContent: path })));
    result.replaceChildren(message, paths);
  } catch (error) { showMessage(String(error).replace(/^Error:\s*/, ""), "error"); }
  finally { analyzeButton.disabled = false; }
});

document.addEventListener("keydown", (event) => {
  if (!event.altKey || event.ctrlKey || event.metaKey) return;
  const key = event.key.toLowerCase();
  if (key === "o") document.querySelector("#choose-source").click();
  if (key === "p") document.querySelector("#choose-output").click();
  if (key === "r") analyzeButton.click();
});
