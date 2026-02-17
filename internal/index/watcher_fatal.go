package index

func (w *Watcher) fatal(err error) {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	onError := w.onError
	w.mu.Unlock()

	if onError != nil {
		onError(err)
	}
}
