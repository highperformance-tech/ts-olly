package fileid

func Query(path string) (uint64, error) {
	return queryFilenameById(path)
}
