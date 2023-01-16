package require

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	js "github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/zbysir/gojsx/pkg/timetrack"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
)

type ModuleLoader func(*js.Runtime, *js.Object)

// SourceLoader represents a function that returns a file data at a given path.
// The function should return ModuleFileDoesNotExistError if the file either doesn't exist or is a directory.
// This error will be ignored by the resolver and the search will continue. Any other errors will be propagated.
type SourceLoader func(path string) ([]byte, error)

type ErrorInvalidModule struct {
	Name string
}

func (e ErrorInvalidModule) Error() string {
	return fmt.Sprintf("Invalid module: '%v'", e.Name)
}

var (
	InvalidModuleError     = errors.New("Invalid module")
	IllegalModuleNameError = errors.New("Illegal module name")

	ModuleFileDoesNotExistError = errors.New("module file does not exist")
)

var native map[string]ModuleLoader

// Registry contains a cache of compiled modules which can be used by multiple Runtimes
type Registry struct {
	sync.Mutex
	native        map[string]ModuleLoader
	compliedCache *lru.Cache[string, *js.Program]
	SrcLoader     SourceLoader
	globalFolders []string
	timeTracker   *timetrack.TimeTracker
}

type RequireModule struct {
	r            *Registry
	runtime      *js.Runtime
	modulesCache *lru.Cache[string, *js.Object]
	nodeModules  map[string]*js.Object
}

func NewRegistry(opts ...Option) *Registry {
	c, err := lru.New[string, *js.Program](100)
	if err != nil {
		panic(err)
	}
	r := &Registry{
		compliedCache: c,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func NewRegistryWithLoader(srcLoader SourceLoader) *Registry {
	return NewRegistry(WithLoader(srcLoader))
}

type Option func(*Registry)

// WithLoader sets a function which will be called by the require() function in order to get a source code for a
// module at the given path. The same function will be used to get external source maps.
// Note, this only affects the modules loaded by the require() function. If you need to use it as a source map
// loader for code parsed in a different way (such as runtime.RunString() or eval()), use (*Runtime).SetParserOptions()
func WithLoader(srcLoader SourceLoader) Option {
	return func(r *Registry) {
		r.SrcLoader = srcLoader
	}
}

// WithGlobalFolders appends the given paths to the registry's list of
// global folders to search if the requested module is not found
// elsewhere.  By default, a registry's global folders list is empty.
// In the reference Node.js implementation, the default global folders
// list is $NODE_PATH, $HOME/.node_modules, $HOME/.node_libraries and
// $PREFIX/lib/node, see
// https://nodejs.org/api/modules.html#modules_loading_from_the_global_folders.
func WithGlobalFolders(globalFolders ...string) Option {
	return func(r *Registry) {
		r.globalFolders = globalFolders
	}
}

// Enable adds the require() function to the specified runtime.
func (r *Registry) Enable(runtime *js.Runtime) *RequireModule {
	c, _ := lru.New[string, *js.Object](100)
	rrt := &RequireModule{
		r:            r,
		runtime:      runtime,
		modulesCache: c,
		nodeModules:  make(map[string]*js.Object),
	}

	runtime.Set("require", rrt.require)
	return rrt
}

func (r *Registry) RegisterNativeModule(name string, loader ModuleLoader) {
	r.Lock()
	defer r.Unlock()

	if r.native == nil {
		r.native = make(map[string]ModuleLoader)
	}
	name = filepathClean(name)
	r.native[name] = loader
}

// DefaultSourceLoader is used if none was set (see WithLoader()). It simply loads files from the host's filesystem.
func DefaultSourceLoader(filename string) ([]byte, error) {
	fp := filepath.FromSlash(filename)
	data, err := os.ReadFile(fp)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, syscall.EISDIR) {
			err = ModuleFileDoesNotExistError
		} else if runtime.GOOS == "windows" {
			if errors.Is(err, syscall.Errno(0x7b)) { // ERROR_INVALID_NAME, The filename, directory name, or volume label syntax is incorrect.
				err = ModuleFileDoesNotExistError
			} else {
				// temporary workaround for https://github.com/dop251/goja_nodejs/issues/21
				fi, err1 := os.Stat(fp)
				if err1 == nil && fi.IsDir() {
					err = ModuleFileDoesNotExistError
				}
			}
		}
	}
	return data, err
}

func (r *Registry) getSource(p string) ([]byte, error) {
	srcLoader := r.SrcLoader
	if srcLoader == nil {
		srcLoader = DefaultSourceLoader
	}
	return srcLoader(p)
}

func (r *Registry) ClearCompliedCache() {
	r.compliedCache.Purge()
}

func mD5(v []byte) string {
	m := md5.New()
	m.Write(v)
	return hex.EncodeToString(m.Sum(nil))
}

func (r *Registry) getCompiledSource(p string) (*js.Program, error) {
	r.Lock()
	defer r.Unlock()

	end := r.timeTracker.Start("getSource")
	buf, err := r.getSource(p)
	end()
	if err != nil {
		return nil, err
	}

	bodyMd5 := mD5(buf)
	prg, ok := r.compliedCache.Get(bodyMd5)
	if ok {
		return prg, nil
	}

	s := string(buf)

	source := "(function(exports, require, module) {" + s + "\n})"
	parsed, err := js.Parse(p, source, parser.WithSourceMapLoader(r.SrcLoader))
	if err != nil {
		return nil, err
	}
	prg, err = js.CompileAST(parsed, false)
	if err == nil {
		r.compliedCache.Add(bodyMd5, prg)
	}
	return prg, err
}

func (r *RequireModule) require(call js.FunctionCall) js.Value {
	ret, err := r.Require(call.Argument(0).String())
	if err != nil {
		if _, ok := err.(*js.Exception); !ok {
			panic(r.runtime.NewGoError(err))
		}
		panic(err)
	}
	return ret
}

func filepathClean(p string) string {
	return path.Clean(p)
}

// Require can be used to import modules from Go source (similar to JS require() function).
func (r *RequireModule) Require(p string) (ret js.Value, err error) {
	module, err := r.resolve(p)
	if err != nil {
		return
	}
	ret = module.Get("exports")
	return
}

func (r *RequireModule) Clean() {
	r.modulesCache.Purge()
	return
}

func Require(runtime *js.Runtime, name string) js.Value {
	if r, ok := js.AssertFunction(runtime.Get("require")); ok {
		mod, err := r(js.Undefined(), runtime.ToValue(name))
		if err != nil {
			panic(err)
		}
		return mod
	}
	panic(runtime.NewTypeError("Please enable require for this runtime using new(require.Registry).Enable(runtime)"))
}

func RegisterNativeModule(name string, loader ModuleLoader) {
	if native == nil {
		native = make(map[string]ModuleLoader)
	}
	name = filepathClean(name)
	native[name] = loader
}
