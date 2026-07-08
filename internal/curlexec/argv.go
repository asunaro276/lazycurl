package curlexec

import (
	"encoding/base64"
	"strconv"
	"time"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// buildArgs constructs the curl argv for req. bodyFile, headerFile, and
// outFile are paths to temp files curl will read the request body from and
// write response headers/body to, respectively. bodyFile is "" when the
// request has no body.
func buildArgs(req httpfile.Request, bodyFile, headerFile, outFile string) []string {
	args := []string{"curl", "-X", method(req.Method), req.URL}

	if authHeader, ok := authorizationHeader(req.Auth); ok {
		args = append(args, "-H", authHeader.Key+": "+authHeader.Value)
	}
	for _, h := range req.Headers {
		if !h.Enabled {
			continue
		}
		args = append(args, "-H", h.Key+": "+h.Value)
	}

	if req.Pragmas.Insecure {
		args = append(args, "-k")
	}
	if seconds, ok := timeoutSeconds(req.Pragmas.Timeout); ok {
		args = append(args, "--max-time", seconds)
	}
	if !req.Pragmas.NoRedirect {
		args = append(args, "-L")
	}

	if bodyFile != "" {
		args = append(args, "--data-binary", "@"+bodyFile)
	}

	args = append(args, "-D", headerFile, "-o", outFile, "-w", "%{json}")
	return args
}

func method(m string) string {
	if m == "" {
		return "GET"
	}
	return m
}

func timeoutSeconds(d string) (string, bool) {
	if d == "" {
		return "", false
	}
	dur, err := time.ParseDuration(d)
	if err != nil {
		return "", false
	}
	return strconv.FormatFloat(dur.Seconds(), 'f', -1, 64), true
}

func authorizationHeader(a httpfile.Auth) (httpfile.KV, bool) {
	switch a.Type {
	case httpfile.AuthBearer:
		if a.Token == "" {
			return httpfile.KV{}, false
		}
		return httpfile.KV{Key: "Authorization", Value: "Bearer " + a.Token}, true
	case httpfile.AuthBasic:
		if a.Username == "" && a.Password == "" {
			return httpfile.KV{}, false
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(a.Username + ":" + a.Password))
		return httpfile.KV{Key: "Authorization", Value: "Basic " + encoded}, true
	default:
		return httpfile.KV{}, false
	}
}
