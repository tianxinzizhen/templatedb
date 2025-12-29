package load

import (
	"embed"
	"runtime"
	"strings"
)

func getCurrentPackageName() string {
	// 获取当前函数的调用栈信息
	pc, _, _, ok := runtime.Caller(3)
	if !ok {
		return ""
	}
	funcInfo := runtime.FuncForPC(pc)
	if funcInfo == nil {
		return ""
	}
	fullName := funcInfo.Name()
	lastDotIndex := strings.LastIndex(fullName, ".")
	if lastDotIndex == -1 {
		return fullName
	}
	return fullName[:lastDotIndex]
}

type LoadFuncDataInfo struct {
	sqlDataInfos map[string][]*SqlDataInfo
}

func NewLoadFuncDataInfo() *LoadFuncDataInfo {
	return &LoadFuncDataInfo{
		sqlDataInfos: make(map[string][]*SqlDataInfo),
	}
}

func (lfi *LoadFuncDataInfo) LoadFuncDataInfoBytes(sqlComments []byte) error {
	pkgName := getCurrentPackageName()
	infos, err := loadCommentBytes(pkgName, sqlComments)
	if err != nil {
		return err
	}
	for _, v := range infos {
		lfi.sqlDataInfos[v.TypeName] = append(lfi.sqlDataInfos[v.TypeName], v)
	}
	return nil
}

func (lfi *LoadFuncDataInfo) LoadFuncDataInfoString(sqlComments string) error {
	pkgName := getCurrentPackageName()
	infos, err := loadCommentString(pkgName, sqlComments)
	if err != nil {
		return err
	}
	for _, v := range infos {
		lfi.sqlDataInfos[v.TypeName] = append(lfi.sqlDataInfos[v.TypeName], v)
	}
	return nil
}

func (lfi *LoadFuncDataInfo) LoadFuncDataInfo(sqlDir embed.FS) error {
	pkgName := getCurrentPackageName()
	dirName := ""
	files, err := sqlDir.ReadDir(".")
	if err != nil {
		return err
	}
	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			dirName = fileInfo.Name() + "/"
			continue
		}
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), ".go") {
			bytes, err := sqlDir.ReadFile(dirName + fileInfo.Name())
			if err != nil {
				return err
			}
			infos, err := loadCommentBytes(pkgName, bytes)
			if err != nil {
				return err
			}
			for _, v := range infos {
				lfi.sqlDataInfos[v.TypeName] = append(lfi.sqlDataInfos[v.TypeName], v)
			}
		}
	}
	return nil
}

func (lfi *LoadFuncDataInfo) GetSqlDataInfo(typeName string) []*SqlDataInfo {
	return lfi.sqlDataInfos[typeName]
}
