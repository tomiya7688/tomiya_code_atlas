package process

//	{
//	  責務: [analysisOperationIDs: 配布CLIで公開する操作の安定した順序を保持する]
//	}
var analysisOperationIDs = []string{
	"call-graph",
	"class-diagram",
	"comments",
	"static-spec",
	"design-quality",
}

//	{
//	  責務: [SupportedOperationIDs: CLI helpと検証が使う操作IDの複製を返す]
//	  戻り値: [[]string: 表示順を保ち、内部状態を変更しない操作ID一覧]
//	}
func SupportedOperationIDs() []string {
	return append([]string(nil), analysisOperationIDs...)
}

//	{
//	  責務: [IsSupportedOperation: 配布CLIがoperationを実行できるか判定する]
//	  引数: [operation: 検証するoperation ID]
//	  戻り値: [bool: operationがregistryにある場合true]
//	}
func IsSupportedOperation(operation string) bool {
	for _, supportedOperation := range analysisOperationIDs {
		if operation == supportedOperation {
			return true
		}
	}
	return false
}
