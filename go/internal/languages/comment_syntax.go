package languages

import (
	"fmt"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

//	{
//	  責務: [CommentSyntax: 言語別の行comment prefixを使って候補をsource copyへ挿入する]
//	  状態: [linePrefix: commentとして挿入する言語固有prefix]
//	}
type CommentSyntax struct {
	linePrefix string
}

//	{
//	  責務: [CommentSyntaxFor: language IDに対応する行comment syntaxを返す]
//	  引数: [languageID: parser contractのlanguage ID]
//	  戻り値: [CommentSyntax: 対応するline comment syntax, error: 対応しないlanguage ID]
//	}
func CommentSyntaxFor(languageID string) (CommentSyntax, error) {
	switch languageID {
	case "python", "gdscript":
		return CommentSyntax{linePrefix: "#"}, nil
	case "csharp", "cpp", "go", "java":
		return CommentSyntax{linePrefix: "//"}, nil
	default:
		return CommentSyntax{}, fmt.Errorf("commentを挿入できない言語です: %s", languageID)
	}
}

//	{
//	  責務: [Insert: Common IRの挿入行へ候補commentを追加したsource copyを作る]
//	  処理: [1: 候補行とtextを検証する, 2: 元の改行とindentを保って行commentを挿入する]
//	  引数: [source: 元fileのtext, candidates: 1始まりの挿入行とcomment text]
//	  戻り値: [string: commentを追加したsource copy, error: 行またはcomment textが不正な場合の理由]
//	}
func (syntax CommentSyntax) Insert(source string, candidates []models.CommentCandidate) (string, error) {
	//	{
	//	  処理: [候補が空ならsource copyへ元textをそのまま返す]
	//	}
	if len(candidates) == 0 {
		return source, nil
	}
	lines := strings.SplitAfter(source, "\n")
	commentsByLine := make(map[int][]models.CommentCandidate, len(candidates))
	//	{
	//	  処理: [全候補の位置とtextを先に検証し、不正なsource copyを返さない]
	//	}
	for _, candidate := range candidates {
		if candidate.Line < 1 || candidate.Line > len(lines) {
			return "", fmt.Errorf("comment挿入行がsource範囲外です: %s line=%d", candidate.Name, candidate.Line)
		}
		if strings.TrimSpace(lines[candidate.Line-1]) == "" {
			return "", fmt.Errorf("comment挿入行が空行です: %s line=%d", candidate.Name, candidate.Line)
		}
		if candidate.Text == "" || strings.ContainsAny(candidate.Text, "\r\n") {
			return "", fmt.Errorf("comment textが空か複数行です: %s", candidate.Name)
		}
		commentsByLine[candidate.Line] = append(commentsByLine[candidate.Line], candidate)
	}

	var output strings.Builder
	output.Grow(len(source) + len(candidates)*48)
	//	{
	//	  処理: [元の行と改行を順に写し、各宣言の直前へlanguage commentを置く]
	//	}
	for lineNumber, sourceLine := range lines {
		for _, candidate := range commentsByLine[lineNumber+1] {
			lineEnding := "\n"
			if strings.HasSuffix(sourceLine, "\r\n") {
				lineEnding = "\r\n"
			}
			fmt.Fprintf(&output, "%s%s %s%s", candidate.Indent, syntax.linePrefix, candidate.Text, lineEnding)
		}
		output.WriteString(sourceLine)
	}
	return output.String(), nil
}
