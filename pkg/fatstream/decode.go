package fatstream

import "github.com/example/prrject-fatbaby/internal/tcpstreamsdk"

func Decode[T any](evt Event) (T, error) { return tcpstreamsdk.Decode[T](evt) }
