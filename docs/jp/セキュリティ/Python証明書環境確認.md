# Python証明書環境の確認記録

## 背景と目的

Issue #254の対応では、依存packageを更新するだけでなく、既存のPython環境とPyInstaller配布物が参照するCA trust storeを確認します。この記録は2026-11-07に行った確認対象、実行方法、結果を残します。

## 確認方法

各repository worktreeの`.venv\Scripts\python.exe`で次を実行し、Python・pip・certifiのversion、`certifi.where()`、pip vendored certifi、Windowsのsystem trust storeを確認します。

```powershell
.\.venv\Scripts\python.exe tools\verify_python_tls.py
```

配布物は全体buildの前にPython OneDir helperをクリーンbuildし、同梱bundleを検査します。

```bat
scripts\dev\pull_request.bat
```

`issue-197-evaluator`の`.venv`は`issue-196-resume\.venv`へのjunctionなので、環境数には重複して数えません。

## 既存worktreeの結果

以下の9環境すべてでCPython 3.14.10、pip 26.3、certifi 2026.10.23、pip vendored certifi 2026.10.23を確認しました。各`certifi.where()`は表の実ファイルを指し、checkerは同bundleのCA名とfingerprintを検査しました。Windows system trust storeは各環境で372 CA名を返し、影響を受けるrootは見つかりませんでした。

| 環境                                          | `certifi.where()`                                                                           | 結果         |
| --------------------------------------------- | ------------------------------------------------------------------------------------------- | ------------ |
| repository root                               | `.venv\Lib\site-packages\certifi\cacert.pem`                                                | 対象rootなし |
| `.build\clean-build-174`                      | `.build\clean-build-174\.venv\Lib\site-packages\certifi\cacert.pem`                         | 対象rootなし |
| `.build\issue-196-resume`（Issue #197と共有） | `.build\issue-196-resume\.venv\Lib\site-packages\certifi\cacert.pem`                        | 対象rootなし |
| `.build\issue-254-python-tls`                 | `.build\issue-254-python-tls\.venv\Lib\site-packages\certifi\cacert.pem`                    | 対象rootなし |
| `issue-210-sequence-recursion` worktree       | `issue-210-sequence-recursion\tomiya_code_atlas\.venv\Lib\site-packages\certifi\cacert.pem` | 対象rootなし |
| `issue-220-sequence-partition` worktree       | `issue-220-sequence-partition\tomiya_code_atlas\.venv\Lib\site-packages\certifi\cacert.pem` | 対象rootなし |
| `issue-221-uml-audit` worktree                | `issue-221-uml-audit\tomiya_code_atlas\.venv\Lib\site-packages\certifi\cacert.pem`          | 対象rootなし |
| `issue-246-parser-layout` worktree            | `issue-246-parser-layout\tomiya_code_atlas\.venv\Lib\site-packages\certifi\cacert.pem`      | 対象rootなし |
| `issue-247-root-files` worktree               | `issue-247-root-files\tomiya_code_atlas\.venv\Lib\site-packages\certifi\cacert.pem`         | 対象rootなし |

worktreeの識別名は`git worktree list`のpath末尾です。各環境のpip vendored certifi fileは同じ相対path `.venv\Lib\site-packages\pip\_vendor\certifi\cacert.pem`（clone worktreeでは`\.venv\Lib\site-packages\pip\_vendor\certifi\cacert.pem`）です。checkerはこのfileのCA名・fingerprintも検査しました。

併せて、repository外のCPython 3.10.11、3.13.13、3.14.7を確認しました。いずれもpip 26.2.2とcertifi 2026.10.23でしたが、このrepositoryが要求するCPython patch floorを満たさないか、対応対象外の系列です。これらはrepositoryのbuildに使わず、setupはpackage取得前に拒否します。

## PyInstaller OneDirと実チェーン

`.build\dist\backends\tomiya-python-backend\_internal\certifi\cacert.pem`を含め、再buildしたOneDirのPython helperを起動してJSON parser contractのsmoke testを通しました。checkerは同じ同梱CA fileを特定し、対象rootが含まれないことを確認しました。

また、公式incident checkerに掲載された実TLS endpointへ接続し、certifi、pip vendored certifi、OneDirに同梱されたcertifi、Windows system trust storeの各trust sourceでTLS handshakeが拒否されることを確認しました。OpenSSLはverify code 20（`unable to get local issuer certificate`）を返しました。

Python helperはJSON parse以外にネットワーク接続を行いません。そのため、製品parserに診断用ネットワーク機能を加えず、配布物に実際に含まれるCA fileと実際に接続したendpointでtrust判定を検証しました。GitHub Releasesと追跡済み`.exe` / `.zip` / `.whl`はなく、古い公開配布物の差し替え対象はありません。worktree外に残る古いignored build出力は今回の検証対象に含めず、安全な成果物として扱いません。

## 再確認時の注意

利用者はrepository rootで`scripts\dev\pull_request.bat`を実行してください。Mavenが自動解決されない環境では、`scripts\build\resolve_maven.ps1`を実行し、返された`mvn.cmd`の`bin` directoryとJDK 25の`bin` directoryを`PATH`へ追加してから再実行してください。NuGet vulnerability metadataの取得が失敗した場合は、`NU1900`を記録し、cached restore/buildの成否と区別してください。
