package drive

type HandleCache interface {
	Ref()
	Unref()
}
