package models

//	{
//	  責務: [CommentTarget: comment候補を適用できるsource宣言の種類を識別する]
//	  値: [class: class宣言; function: function宣言; method: method宣言]
//	}
type CommentTarget string

//	{
//	  責務: [CommentTarget constants: Common IR entity kindと一致するcomment対象値を定義する]
//	}
const (
	CommentTargetClass    CommentTarget = "class"
	CommentTargetFunction CommentTarget = "function"
	CommentTargetMethod   CommentTarget = "method"
)

//	{
//	  責務: [CommentCandidate: sourceを書き換えずに提案するcommentと挿入位置を保持する]
//	  フィールド: [Target: comment対象の宣言種別; Name: 宣言名; Line: 1始まりの挿入行; Indent: 宣言位置に合わせるindent; Text: 生成する日本語comment]
//	}
type CommentCandidate struct {
	Target CommentTarget `json:"target"`
	Name   string        `json:"name"`
	Line   int           `json:"line"`
	Indent string        `json:"indent"`
	Text   string        `json:"text"`
}
