package dara

// Plan 9 不提供 POSIX socket errno；保留通用超时和 EOF 分类。
func transportErrnoCode(err error) string {
	return ""
}
