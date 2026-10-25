package ui

import (
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

//	{
//	  責務: [layout: 利用者の操作を処理し、入力から解析結果までを一画面に表示する,],
//	  処理: [1: 画面全体でショートカットを受け付ける, 2: ボタン操作を処理する, 3: 入力・図の種類・保存先・状態・結果の順に配置する,],
//	}
func (application *desktopApp) layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	event.Op(gtx.Ops, &application.shortcutTarget)
	application.handleShortcuts(gtx)
	application.handleButtons(gtx)
	return layout.Inset{
		Top: unit.Dp(16), Bottom: unit.Dp(14), Left: unit.Dp(20), Right: unit.Dp(20),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, material.H4(theme, windowTitle).Layout)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Body2(theme, "Go版で選べる解析をまとめて実行し、結果を指定フォルダーへ保存します。").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return application.layoutInput(gtx, theme)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return application.layoutOperations(gtx, theme)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return application.layoutOutputDirectory(gtx, theme)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					label := "解析を実行（Alt+R）"
					if application.analysisPending {
						label = "解析中です…"
					}
					return material.Button(theme, &application.runButton, label).Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return application.layoutStatus(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Subtitle2(theme, "生成したファイルの場所").Layout(gtx)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return application.layoutOutputPaths(gtx, theme)
			}),
		)
	})
}

//	{
//	  責務: [layoutInput: 解析対象のパス入力欄とOS標準のファイル・フォルダー選択ボタンを表示する,],
//	}
func (application *desktopApp) layoutInput(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Subtitle1(theme, "① 解析するファイルまたはプロジェクトフォルダー").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(theme, &application.sourceEditor, "パスを入力するか、右のボタンで選んでください").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Button(theme, &application.chooseFileButton, "ファイルを選ぶ（Alt+F）").Layout(gtx)
				}),
				layout.Rigid(horizontalGap(unit.Dp(8))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Button(theme, &application.chooseInputButton, "フォルダーを選ぶ（Alt+P）").Layout(gtx)
				}),
			)
		}),
	)
}

//	{
//	  責務: [layoutOperations: Processが対応する図を複数選べるチェックボックスで表示する,],
//	}
func (application *desktopApp) layoutOperations(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Subtitle1(theme, "② 作りたい図（複数選択できます）").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Caption(theme, "現在のGo版で利用できる図だけを表示します。").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(application.operations))
			for index := range application.operations {
				operation := &application.operations[index]
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.CheckBox(theme, &operation.checked, operation.definition.label).Layout(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
		}),
	)
}

//	{
//	  責務: [layoutOutputDirectory: 生成したMarkdownファイルの保存先を入力・選択できる欄を表示する,],
//	}
func (application *desktopApp) layoutOutputDirectory(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Subtitle1(theme, "③ 結果の保存先フォルダー").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return material.Editor(theme, &application.outputEditor, "保存先フォルダーのパスを入力してください").Layout(gtx)
					}),
					layout.Rigid(horizontalGap(unit.Dp(8))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Button(theme, &application.chooseOutputButton, "保存先を選ぶ（Alt+O）").Layout(gtx)
					}),
				)
			})
		}),
	)
}

//	{
//	  責務: [layoutStatus: 実行状況とProcess・構文解析器から返った診断を分けて表示する,],
//	}
func (application *desktopApp) layoutStatus(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(theme, "状態: "+application.status).Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if application.diagnostic == "" {
				return layout.Dimensions{}
			}
			style := material.Body2(theme, "診断: "+application.diagnostic)
			style.Color = color.NRGBA{R: 150, G: 28, B: 28, A: 255}
			return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(8)}.Layout(gtx, style.Layout)
		}),
	)
}

//	{
//	  責務: [layoutOutputPaths: Processが保存したファイルのパスをスクロール可能な一覧で表示する,],
//	}
func (application *desktopApp) layoutOutputPaths(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if len(application.outputPaths) == 0 {
		return material.Body2(theme, "解析後、保存したファイルの場所をここに表示します。").Layout(gtx)
	}
	application.outputList.Axis = layout.Vertical
	return material.List(theme, &application.outputList).Layout(gtx, len(application.outputPaths), func(gtx layout.Context, index int) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3)}.Layout(gtx, material.Body2(theme, application.outputPaths[index]).Layout)
	})
}

//	{
//	  責務: [handleButtons: ボタン操作をファイル選択またはProcessの実行へ振り分ける,],
//	}
func (application *desktopApp) handleButtons(gtx layout.Context) {
	if application.chooseFileButton.Clicked(gtx) {
		application.startPicker(pickerSourceFile)
	}
	if application.chooseInputButton.Clicked(gtx) {
		application.startPicker(pickerSourceFolder)
	}
	if application.chooseOutputButton.Clicked(gtx) {
		application.startPicker(pickerOutputFolder)
	}
	if application.runButton.Clicked(gtx) {
		application.startAnalysis()
	}
}

//	{
//	  責務: [handleShortcuts: 画面に示したAlt+F/P/O/Rを対応するボタン操作へ接続する,],
//	}
func (application *desktopApp) handleShortcuts(gtx layout.Context) {
	application.handleShortcut(gtx, "F", func() { application.startPicker(pickerSourceFile) })
	application.handleShortcut(gtx, "P", func() { application.startPicker(pickerSourceFolder) })
	application.handleShortcut(gtx, "O", func() { application.startPicker(pickerOutputFolder) })
	application.handleShortcut(gtx, "R", application.startAnalysis)
}

//	{
//	  責務: [handleShortcut: Altキーと指定した英字キーの組み合わせを処理する,],
//	  引数: [gtx: Gioのイベント情報, name: 判定する英字キー, action: キーを押したときに実行する操作,],
//	}
func (application *desktopApp) handleShortcut(gtx layout.Context, name string, action func()) {
	for {
		routedEvent, ok := gtx.Event(key.Filter{Required: key.ModAlt, Name: key.Name(name)})
		if !ok {
			return
		}
		keyEvent, ok := routedEvent.(key.Event)
		if ok && keyEvent.State == key.Press {
			action()
		}
	}
}

//	{
//	  責務: [horizontalGap: 同じ行の画面部品の間に指定幅の空白を作る,],
//	  引数: [width: 空ける幅,],
//	  戻り値: [layout.Widget: 指定幅の空白を配置する関数,],
//	}
func horizontalGap(width unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Spacer{Width: width}.Layout(gtx)
	}
}
