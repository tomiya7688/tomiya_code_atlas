//	{
//	  責務: [staticdoc: 静的仕様書の生成とMarkdown描画を調整する互換入口]
//	}
package staticdoc

import (
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
	"github.com/tomiya7688/tomiya_code_atlas/internal/renderers"
)

//	{
//	  責務: [Render: Common IR moduleを静的仕様書Markdownへ変換する]
//	  引数: [module: 解析済みCommon IR; sourceName: 表示するsource名]
//	  戻り値: [string: 静的仕様書Markdown]
//	}
func Render(module commonir.Module, sourceName string) string {
	logicalDocument := generators.BuildStaticSpecification(module, sourceName)
	return renderers.RenderStaticSpecificationMarkdown(logicalDocument)
}
