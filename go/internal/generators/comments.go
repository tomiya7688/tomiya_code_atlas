package generators

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

//	{
//	  責務: [commentCamelCaseBoundary: lower/digitとuppercaseの間へword separatorを入れるpatternを保持する]
//	}
var commentCamelCaseBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

//	{
//	  責務: [GenerateCommentCandidates: Common IRの宣言factからsource非変更の日本語comment候補を生成する]
//	  処理: [1: class/function/methodだけを選ぶ; 2: 既存docstringと空の名前を除く; 3: 宣言名とcall factから候補textを作る]
//	  引数: [module: language adapterが正規化したCommon IR。parser固有型を含まない]
//	  戻り値: [1: []models.CommentCandidate: 宣言順を保つ候補一覧; 2: error: 位置や宣言名が不正な場合の理由]
//	}
func GenerateCommentCandidates(module commonir.Module) ([]models.CommentCandidate, error) {
	candidates := make([]models.CommentCandidate, 0)
	for _, entity := range module.Entities {
		target, supported := commentTarget(entity.Kind)
		if !supported || hasExistingDocumentation(entity.Docstring) {
			continue
		}
		if entity.Name == "" {
			return nil, fmt.Errorf("comment候補の宣言名を空にできません")
		}
		if entity.Line < 1 || entity.Indent < 0 {
			return nil, fmt.Errorf("%sのcomment位置が不正です: line=%d indent=%d", entity.Name, entity.Line, entity.Indent)
		}
		candidates = append(candidates, models.CommentCandidate{
			Target: target,
			Name:   entity.Name,
			Line:   entity.Line,
			Indent: strings.Repeat(" ", entity.Indent),
			Text:   describeComment(entity, target),
		})
	}
	return candidates, nil
}

//	{
//	  責務: [commentTarget: Common IRのentity kindをcomment対象種別へ対応付ける]
//	  処理: [1: 対応する宣言種別を返す; 2: 対応外のkindは対象外として返す]
//	  引数: [kind: Common IR entityのkind]
//	  戻り値: [1: models.CommentTarget: comment対象種別; 2: bool: 対応種別ならtrue]
//	}
func commentTarget(kind string) (models.CommentTarget, bool) {
	switch kind {
	case "class":
		return models.CommentTargetClass, true
	case "function":
		return models.CommentTargetFunction, true
	case "method":
		return models.CommentTargetMethod, true
	default:
		return "", false
	}
}

//	{
//	  責務: [hasExistingDocumentation: 宣言に既存docstringがあるか判定する]
//	  処理: [1: nilを除く; 2: 空文字以外を既存documentationとして扱う]
//	  引数: [docstring: Common IRのoptional docstring]
//	  戻り値: [bool: 既存documentationがあればtrue]
//	}
func hasExistingDocumentation(docstring *string) bool {
	return docstring != nil && *docstring != ""
}

//	{
//	  責務: [describeComment: 宣言種別・名前・観測callから説明文を作る]
//	  処理: [1: 宣言名を読みやすい語列へ分ける; 2: classと既知動詞を説明する; 3: その他はcall有無に応じた文を返す]
//	  引数: [entity: 名前とcall factを持つCommon IR entity; target: 対象種別]
//	  戻り値: [string: 宣言順序とsource構文に依存しない日本語comment text]
//	}
func describeComment(entity commonir.Entity, target models.CommentTarget) string {
	words := commentWords(entity.Name)
	readableName := strings.Join(words, " ")
	if readableName == "" {
		readableName = entity.Name
	}
	if target == models.CommentTargetClass {
		return readableName + "に関する状態と処理をまとめます。"
	}
	verb := entity.Name
	if len(words) > 0 {
		verb = words[0]
	}
	subject := "処理"
	if len(words) > 1 {
		subject = strings.Join(words[1:], " ")
	}
	if text := commentVerbText(verb, subject); text != "" {
		return text
	}
	if len(entity.Calls) > 0 {
		return readableName + "の処理を実行し、関連処理を連携させます。"
	}
	return readableName + "の処理を行います。"
}

//	{
//	  責務: [commentVerbText: 既知の宣言名動詞を日本語の利用者向け文へ変換する]
//	  処理: [1: 動詞ごとに自然な述語を選ぶ; 2: 未知の動詞は空文字で返す]
//	  引数: [verb: 宣言名の先頭語; subject: 宣言名から抽出した対象]
//	  戻り値: [string: 対応する日本語説明。未知のverbは空文字]
//	}
func commentVerbText(verb, subject string) string {
	switch verb {
	case "get":
		return subject + "を取得します。"
	case "load":
		return subject + "を読み込みます。"
	case "read":
		return subject + "を読み取ります。"
	case "parse":
		return subject + "を解析します。"
	case "create", "generate":
		return subject + "を生成します。"
	case "build":
		return subject + "を構築します。"
	case "render":
		return subject + "を描画用表現へ変換します。"
	case "save":
		return subject + "を保存します。"
	case "write":
		return subject + "を書き出します。"
	case "update":
		return subject + "を更新します。"
	case "delete":
		return subject + "を削除します。"
	case "remove":
		return subject + "を取り除きます。"
	case "validate":
		return subject + "を検証します。"
	case "check":
		return subject + "を確認します。"
	case "find":
		return subject + "を検索します。"
	case "collect":
		return subject + "を収集します。"
	case "convert":
		return subject + "を変換します。"
	case "run":
		return subject + "の処理を実行します。"
	default:
		return ""
	}
}

//	{
//	  責務: [commentWords: snake_caseとcamelCaseの宣言名を語列へ分割する]
//	  処理: [1: camelCase境界へunderscoreを挿入する; 2: underscoreで分けて小文字化する]
//	  引数: [name: Common IR entity name]
//	  戻り値: [[]string: 空語を除いた小文字語列]
//	}
func commentWords(name string) []string {
	normalized := commentCamelCaseBoundary.ReplaceAllString(name, "${1}_${2}")
	words := strings.Split(strings.Trim(normalized, "_"), "_")
	result := make([]string, 0, len(words))
	for _, word := range words {
		if word != "" {
			result = append(result, strings.ToLower(word))
		}
	}
	return result
}
