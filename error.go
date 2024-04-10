package wintun

type ErrLoaded struct{}

func (ErrLoaded) Error() string   { return "wintun loaded" }
func (ErrLoaded) Temporary() bool { return true }

type ErrNotLoad struct{}

func (ErrNotLoad) Error() string { return "wintun not load" }

type ErrAdapterClosed struct{}

func (ErrAdapterClosed) Error() string { return "adapter closed" }

type ErrAdapterStoped struct{}

func (ErrAdapterStoped) Error() string { return "adapter stoped" }
