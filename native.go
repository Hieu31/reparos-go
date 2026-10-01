package reparos

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/ebitengine/purego"
)

const nativeABIVersion = 2

// errNativeNotFound is returned when no native bridge library can be located.
var errNativeNotFound = errors.New("native bridge library not found")

var nativeStatus = map[int32]string{
	-1: "invalid argument",
	-2: "tokenization failed",
	-3: "translation failed",
	-4: "internal error",
	-5: "input too long",
	-6: "failed to load model",
}

func nativeError(op string, code int32) error {
	if msg, ok := nativeStatus[code]; ok {
		return fmt.Errorf("%s: %s", op, msg)
	}
	return fmt.Errorf("%s: unknown error %d", op, code)
}

// nativeEngine runs CTranslate2 in-process through the C bridge library.
type nativeEngine struct {
	mu     sync.RWMutex // Predict holds RLock, close holds Lock.
	closed bool
	handle uintptr

	predictFn      func(handle uintptr, query *byte, beam, nhyp int32, out *uintptr) int32
	resultCountFn  func(result uintptr) int32
	resultScoresFn func(result uintptr, out *float32, cap int32) int32
	resultTextFn   func(result uintptr, index int32, buf *byte, cap int32) int32
	resultFreeFn   func(result uintptr)
	freeFn         func(handle uintptr)
}

func nativeLibName() string {
	switch runtime.GOOS {
	case "windows":
		return "reparos_bridge.dll"
	case "darwin":
		return "reparos_bridge.dylib"
	default:
		return "reparos_bridge.so"
	}
}

// findNativeLib looks for the bridge library in, in order: the explicit path,
// $REPAROS_NATIVE_LIB, the user cache, next to the executable, and the working dir.
func findNativeLib(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("native library %q: %w", explicit, err)
		}
		return explicit, nil
	}
	if env := os.Getenv("REPAROS_NATIVE_LIB"); env != "" {
		if _, err := os.Stat(env); err != nil {
			return "", fmt.Errorf("REPAROS_NATIVE_LIB %q: %w", env, err)
		}
		return env, nil
	}

	name := nativeLibName()
	var candidates []string
	if cache, err := os.UserCacheDir(); err == nil {
		candidates = append(candidates, filepath.Join(cache, "reparos-go", "native", name))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), name))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, name),
			filepath.Join(wd, "bridge", name),
			filepath.Join(wd, "internal", "native", name))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", errNativeNotFound
}

func cString(s string) []byte {
	b := make([]byte, len(s)+1)
	copy(b, s)
	return b
}

func newNativeEngine(libPath, modelDir string, opts Options) (*nativeEngine, error) {
	lib, err := openLibrary(libPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", libPath, err)
	}

	var (
		abiFn    func() int32
		createFn func(modelDir, device, compute *byte, intra, inter int32, out *uintptr) int32
	)
	e := &nativeEngine{}
	purego.RegisterLibFunc(&abiFn, lib, "reparos_abi_version")
	if v := abiFn(); v != nativeABIVersion {
		return nil, fmt.Errorf("native library ABI version %d, want %d", v, nativeABIVersion)
	}
	purego.RegisterLibFunc(&createFn, lib, "reparos_create")
	purego.RegisterLibFunc(&e.predictFn, lib, "reparos_predict")
	purego.RegisterLibFunc(&e.resultCountFn, lib, "reparos_result_count")
	purego.RegisterLibFunc(&e.resultScoresFn, lib, "reparos_result_scores")
	purego.RegisterLibFunc(&e.resultTextFn, lib, "reparos_result_text")
	purego.RegisterLibFunc(&e.resultFreeFn, lib, "reparos_result_free")
	purego.RegisterLibFunc(&e.freeFn, lib, "reparos_free")

	dir, dev, ct := cString(modelDir), cString(opts.Device), cString(opts.ComputeType)
	if code := createFn(&dir[0], &dev[0], &ct[0], int32(opts.IntraThreads), int32(opts.InterThreads), &e.handle); code != 0 {
		return nil, nativeError("create", code)
	}
	return e, nil
}

func (e *nativeEngine) predict(query string, beamSize, numHypotheses int) (*Result, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.closed {
		return nil, errors.New("predictor is already closed")
	}

	q := cString(query)
	var res uintptr
	if code := e.predictFn(e.handle, &q[0], int32(beamSize), int32(numHypotheses), &res); code != 0 {
		return nil, nativeError("predict", code)
	}
	defer e.resultFreeFn(res)

	n := int(e.resultCountFn(res))
	out := &Result{Hypotheses: make([]string, n), Scores: make([]float64, n)}

	scores := make([]float32, n+1)
	e.resultScoresFn(res, &scores[0], int32(n))
	for i := 0; i < n; i++ {
		out.Scores[i] = float64(scores[i])

		size := e.resultTextFn(res, int32(i), nil, 0)
		if size < 0 {
			return nil, nativeError("result", size)
		}
		buf := make([]byte, size+1)
		e.resultTextFn(res, int32(i), &buf[0], size+1)
		out.Hypotheses[i] = string(buf[:size])
	}

	out.Top1Query = query
	if n > 0 {
		out.Top1Query = out.Hypotheses[0]
	}
	return out, nil
}

func (e *nativeEngine) close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	e.closed = true
	e.freeFn(e.handle)
	e.handle = 0
	return nil
}
