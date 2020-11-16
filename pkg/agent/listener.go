package agent

import (
	"io"
	"io/ioutil"
	"net"
	"strings"

	"github.com/cockroachdb/cmux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

/* PANDARIA: Fix get request head X-RANCHER-GROUP is nil.
we need to be aware that keys are canonicalized header["X-Rancher-Group"] will be work, but header["X-RANCHER-GROUP"] will not return any value.
Details view https://github.com/golang/go/issues/34799
*/
const (
	rancherUserHeaderKey   = "X-Rancher-User"
	rancherGroupHeaderKey  = "X-Rancher-Group"
	authorizationHeaderKey = "Authorization"
)

func createHTTPListener(mux cmux.CMux) net.Listener {
	return mux.Match(
		cmux.HTTP1Fast(),
	)
}

func createGRPCListener(mux cmux.CMux) net.Listener {
	return mux.Match(
		http2HeaderFieldEqual(map[string]string{
			"Content-Type": "application/grpc",
		}),
	)
}

func hasHTTP2Preface(r io.Reader) bool {
	var b [len(http2.ClientPreface)]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return false
	}

	return string(b[:]) == http2.ClientPreface
}

func http2HeaderFieldEqual(nameValuePairs map[string]string) cmux.Matcher {
	return func(r io.Reader) (matched bool) {
		if !hasHTTP2Preface(r) {
			return false
		}

		framer := http2.NewFramer(ioutil.Discard, r)
		hdec := hpack.NewDecoder(uint32(4<<10), func(hf hpack.HeaderField) {
			for name, value := range nameValuePairs {
				matched = strings.EqualFold(hf.Name, name) && hf.Value == value
				if !matched {
					break
				}
			}
		})
		for {
			f, err := framer.ReadFrame()
			if err != nil {
				return false
			}

			switch f := f.(type) {
			case *http2.HeadersFrame:
				if _, err := hdec.Write(f.HeaderBlockFragment()); err != nil {
					return false
				}
				if matched {
					return true
				}

				if f.FrameHeader.Flags&http2.FlagHeadersEndHeaders != 0 {
					return false
				}
			}
		}
	}
}
