package benchmark

import (
	"fmt"
	"os"
)

const (
	cacheStatusDisabled      = "disabled"
	cacheStatusBypassed      = "bypassed"
	cacheStatusApplied       = "applied"
	cacheStatusUnsupported   = "unsupported"
	cacheStatusFailed        = "failed"
	cacheStatusNotApplicable = "not_applicable"
)

func applyCacheControl(file *os.File, config Config) (CacheControlResult, error) {
	result := CacheControlResult{Mode: config.CacheControl}
	if config.CacheControl == "" || config.CacheControl == CacheControlOff {
		result.Mode = CacheControlOff
		result.Status = cacheStatusDisabled
		return result, nil
	}
	if config.IOMode == DirectIO {
		result.API = "direct_io"
		result.Status = cacheStatusBypassed
		result.Detail = "direct I/O already bypasses the normal file cache"
		return result, nil
	}

	api, supported, err := platformCacheControl(file)
	result.API = api
	if !supported {
		result.Status = cacheStatusUnsupported
		result.Detail = "safe per-file cache control is unavailable on this platform"
		if config.CacheControl == CacheControlRequire {
			return result, fmt.Errorf("required cache control is unsupported: %s", result.Detail)
		}
		return result, nil
	}
	if err != nil {
		result.Status = cacheStatusFailed
		result.Detail = err.Error()
		if config.CacheControl == CacheControlRequire {
			return result, fmt.Errorf("required cache control failed using %s: %w", api, err)
		}
		return result, nil
	}
	result.Status = cacheStatusApplied
	return result, nil
}

func cacheControlNotApplicable(config Config, detail string) CacheControlResult {
	return CacheControlResult{
		Mode:   config.CacheControl,
		Status: cacheStatusNotApplicable,
		Detail: detail,
	}
}
