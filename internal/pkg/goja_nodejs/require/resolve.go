package require

import (
	"encoding/json"
	"errors"
	js "github.com/dop251/goja"
	"path"
	"strings"
)

// NodeJS module search algorithm described by
// https://nodejs.org/api/modules.html#modules_all_together
func (r *RequireModule) resolve(modpath string) (module *js.Object, err error) {
	origPath, modpath := modpath, filepathClean(modpath)
	if modpath == "" {
		return nil, IllegalModuleNameError
	}

	module, err = r.loadNative(modpath)
	if err == nil {
		return
	}

	var start string
	err = nil
	if path.IsAbs(origPath) {
		start = "/"
	} else {
		start = r.getCurrentModulePath()
	}

	p := path.Join(start, modpath)
	if strings.HasPrefix(origPath, "./") ||
		strings.HasPrefix(origPath, "/") || strings.HasPrefix(origPath, "../") ||
		origPath == "." || origPath == ".." {
		if module, _ = r.modulesCache.Get(p); module != nil {
			return
		}
		module, err = r.loadAsFileOrDirectory(p)
		if err == nil && module != nil {
			r.modulesCache.Add(p, module)
		}
	} else {
		if module = r.nodeModules[p]; module != nil {
			return
		}
		module, err = r.loadNodeModules(modpath, start)
		if err == nil && module != nil {
			r.nodeModules[p] = module
		}
	}

	if module == nil && err == nil {
		err = &ErrorInvalidModule{Name: p}
	}
	return
}

func (r *RequireModule) loadNative(path string) (*js.Object, error) {
	module, ok := r.modulesCache.Get(path)
	if ok {
		return module, nil
	}

	var ldr ModuleLoader
	if ldr = r.r.native[path]; ldr == nil {
		ldr = native[path]
	}

	if ldr != nil {
		module = r.createModuleObject()
		r.modulesCache.Add(path, module)
		ldr(r.runtime, module)
		return module, nil
	}

	return nil, &ErrorInvalidModule{Name: path}
}

func (r *RequireModule) loadAsFileOrDirectory(path string) (module *js.Object, err error) {
	if module, err = r.loadAsFile(path); module != nil || err != nil {
		return
	}

	return r.loadAsDirectory(path)
}

func (r *RequireModule) loadAsFile(path string) (module *js.Object, err error) {
	if module, err = r.loadModule(path); module != nil || err != nil {
		return
	}

	p := path + ".js"
	if module, err = r.loadModule(p); module != nil || err != nil {
		return
	}

	p = path + ".json"
	return r.loadModule(p)
}

func (r *RequireModule) loadIndex(modpath string) (module *js.Object, err error) {
	p := path.Join(modpath, "index.js")
	if module, err = r.loadModule(p); module != nil || err != nil {
		return
	}

	p = path.Join(modpath, "index.json")
	return r.loadModule(p)
}

func (r *RequireModule) loadAsDirectory(modpath string) (module *js.Object, err error) {
	p := path.Join(modpath, "package.json")
	buf, err := r.r.getSource(p)
	if err != nil {
		return r.loadIndex(modpath)
	}
	var pkg struct {
		Main string
	}
	err = json.Unmarshal(buf, &pkg)
	if err != nil || len(pkg.Main) == 0 {
		return r.loadIndex(modpath)
	}

	m := path.Join(modpath, pkg.Main)
	if module, err = r.loadAsFile(m); module != nil || err != nil {
		return
	}

	return r.loadIndex(m)
}

func (r *RequireModule) loadNodeModule(modpath, start string) (*js.Object, error) {
	return r.loadAsFileOrDirectory(path.Join(start, modpath))
}

func (r *RequireModule) loadNodeModules(modpath, start string) (module *js.Object, err error) {
	for _, dir := range r.r.globalFolders {
		if module, err = r.loadNodeModule(modpath, dir); module != nil || err != nil {
			return
		}
	}
	for {
		var p string
		if path.Base(start) != "node_modules" {
			p = path.Join(start, "node_modules")
		} else {
			p = start
		}
		if module, err = r.loadNodeModule(modpath, p); module != nil || err != nil {
			return
		}
		if start == ".." { // Dir('..') is '.'
			break
		}
		parent := path.Dir(start)
		if parent == start {
			break
		}
		start = parent
	}

	return
}

func (r *RequireModule) getCurrentModulePath() string {
	var buf [2]js.StackFrame
	frames := r.runtime.CaptureCallStack(2, buf[:0])
	if len(frames) < 2 {
		return "."
	}
	return path.Dir(frames[1].SrcName())
}

func (r *RequireModule) createModuleObject() *js.Object {
	module := r.runtime.NewObject()
	module.Set("exports", r.runtime.NewObject())
	return module
}

func (r *RequireModule) loadModule(path string) (*js.Object, error) {
	module, ok := r.modulesCache.Get(path)
	if !ok {
		module = r.createModuleObject()
		// 解决循环引用
		r.modulesCache.Add(path, module)
		end := r.r.timeTracker.Start("loadModuleFile")
		err := r.loadModuleFile(path, module)
		end()
		if err != nil {
			module = nil
			r.modulesCache.Remove(path)
			if errors.Is(err, ModuleFileDoesNotExistError) {
				err = nil
			}
		}
		return module, err
	}
	return module, nil
}

func (r *RequireModule) loadModuleFile(path string, jsModule *js.Object) error {
	end := r.r.timeTracker.Start("getCompiledSource")
	prg, err := r.r.getCompiledSource(path)
	end()

	if err != nil {
		return err
	}

	f, err := r.runtime.RunProgram(prg)
	if err != nil {
		return err
	}

	if call, ok := js.AssertFunction(f); ok {
		jsExports := jsModule.Get("exports")
		jsRequire := r.runtime.Get("require")

		// Run the module source, with "jsExports" as "this",
		// "jsExports" as the "exports" variable, "jsRequire"
		// as the "require" variable and "jsModule" as the
		// "module" variable (Nodejs capable).
		_, err = call(jsExports, jsExports, jsRequire, jsModule)
		if err != nil {
			return err
		}
	} else {
		return &ErrorInvalidModule{Name: path}
	}

	return nil
}
