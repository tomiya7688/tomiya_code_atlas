package data

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//	{
//	  責務: [IsPathWithinOrEqual: candidate pathがroot pathと同じかその配下かを判定する,]
//	  処理: [1: 既存symlinkを解決したabsolute pathを取得する, 2: candidateがrootと同じか配下かを返す,]
//	  引数: [rootPath: 包含元のfolder path, candidatePath: 判定対象のpath。末端folderは未作成でもよい,]
//	  戻り値: [isWithinOrEqual: candidateがrootと同じか配下ならtrue, error: pathを安全に解決できない場合の説明,]
//	}
func IsPathWithinOrEqual(rootPath string, candidatePath string) (bool, error) {
	resolvedRoot, err := resolvePathWithExistingParents(rootPath)
	if err != nil {
		return false, fmt.Errorf("root pathを解決できません (%s): %w", rootPath, err)
	}
	resolvedCandidate, err := resolvePathWithExistingParents(candidatePath)
	if err != nil {
		return false, fmt.Errorf("candidate pathを解決できません (%s): %w", candidatePath, err)
	}
	rootVolume := filepath.VolumeName(resolvedRoot)
	candidateVolume := filepath.VolumeName(resolvedCandidate)
	if !strings.EqualFold(rootVolume, candidateVolume) {
		return false, nil
	}
	relativePath, err := filepath.Rel(resolvedRoot, resolvedCandidate)
	if err != nil {
		return false, fmt.Errorf("相対pathを比較できません: %w", err)
	}
	if relativePath == "." {
		return true, nil
	}
	parentPrefix := ".." + string(filepath.Separator)
	isOutside := relativePath == ".." || strings.HasPrefix(relativePath, parentPrefix) || filepath.IsAbs(relativePath)
	return !isOutside, nil
}

//	{
//	  責務: [resolvePathWithExistingParents: 未作成の末端pathを保ちながら既存symlinkまで解決する,]
//	  処理: [1: pathをabsolute pathへ変換する, 2: 既存ancestorをsymlink解決する, 3: 未作成segmentを戻す,]
//	  引数: [path: 比較用に解決するpath,]
//	  戻り値: [resolvedPath: 既存symlinkを解決したabsolute path, error: 空path・symlink・file systemの確認に失敗した場合の説明,]
//	}
func resolvePathWithExistingParents(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("pathが空です")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("absolute pathを取得できません: %w", err)
	}
	currentPath := filepath.Clean(absolutePath)
	missingSegments := make([]string, 0)
	for {
		resolvedPath, resolveErr := filepath.EvalSymlinks(currentPath)
		if resolveErr == nil {
			for segmentIndex := len(missingSegments) - 1; segmentIndex >= 0; segmentIndex-- {
				resolvedPath = filepath.Join(resolvedPath, missingSegments[segmentIndex])
			}
			return filepath.Clean(resolvedPath), nil
		}
		if !errors.Is(resolveErr, fs.ErrNotExist) {
			return "", fmt.Errorf("既存pathのsymlinkを解決できません (%s): %w", currentPath, resolveErr)
		}
		fileInfo, statErr := os.Lstat(currentPath)
		if statErr == nil && fileInfo.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("dangling symlinkを解決できません (%s)", currentPath)
		}
		if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
			return "", fmt.Errorf("既存pathを確認できません (%s): %w", currentPath, statErr)
		}
		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			return "", fmt.Errorf("既存ancestorが見つかりません (%s)", absolutePath)
		}
		missingSegments = append(missingSegments, filepath.Base(currentPath))
		currentPath = parentPath
	}
}
