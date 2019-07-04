// +build soong

/*
 * Copyright 2019 Arm Limited.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * This file is included when Bob is being run as a Soong plugin.
 *
 * The build tag on the first line ensures that it is not included in the build
 * by accident, and that it is not included in `go test` or similar checks,
 * which would fail, because Soong is not available in that environment.
 */

package core

import (
	"sync"

	"android/soong/android"
	"github.com/google/blueprint"
)

const (
	bobModuleSuffix = "__bob_module_type"
)

var (
	loadConfigOnce   sync.Once
	onceLoadedConfig *bobConfig
)

// During the build, Soong will do a "test" of each plugin, which loads the
// module, including calling its `init()` functions. That means that we can't
// load the config file in `init()`, because the tests would fail if it doesn't
// exist. Work around this by deferring loading the config file until a module
// factory is actually called.
func soongGetConfig() *bobConfig {
	loadConfigOnce.Do(func() {
		onceLoadedConfig = &bobConfig{}
		onceLoadedConfig.Properties = loadConfig(jsonPath)

		if !onceLoadedConfig.Properties.GetBool("builder_soong") {
			panic("Build bootstrapped for Soong, but Soong builder has not been enabled")
		}
		onceLoadedConfig.Generator = &soongGenerator{}
	})
	return onceLoadedConfig

}

func getConfig(interface{}) *bobConfig {
	return soongGetConfig()
}

type moduleBase struct {
	blueprint.SimpleName
	android.ModuleBase
}

func (m *moduleBase) GenerateAndroidBuildActions(ctx android.ModuleContext) {}

// All Bob module types need a Name() function so that the Bob module
// names do not conflict with the names of the soong modules we will
// be creating.
func (m *moduleBase) Name() string {
	return m.SimpleName.Name() + bobModuleSuffix
}

type soongGenerator struct {
	toolchainSet
}

func (g *soongGenerator) aliasActions(m *alias, ctx blueprint.ModuleContext)                        {}
func (g *soongGenerator) binaryActions(*binary, blueprint.ModuleContext)                            {}
func (g *soongGenerator) genBinaryActions(*generateBinary, blueprint.ModuleContext, []inout)        {}
func (g *soongGenerator) genSharedActions(*generateSharedLibrary, blueprint.ModuleContext, []inout) {}
func (g *soongGenerator) genStaticActions(*generateStaticLibrary, blueprint.ModuleContext, []inout) {}
func (g *soongGenerator) generateSourceActions(*generateSource, blueprint.ModuleContext, []inout)   {}
func (g *soongGenerator) kernelModuleActions(m *kernelModule, ctx blueprint.ModuleContext)          {}
func (g *soongGenerator) resourceActions(*resource, blueprint.ModuleContext)                        {}
func (g *soongGenerator) sharedActions(*sharedLibrary, blueprint.ModuleContext)                     {}
func (g *soongGenerator) staticActions(*staticLibrary, blueprint.ModuleContext)                     {}
func (g *soongGenerator) transformSourceActions(*transformSource, blueprint.ModuleContext, []inout) {}

func (g *soongGenerator) buildDir() string                           { return "" }
func (g *soongGenerator) sourcePrefix() string                       { return "" }
func (g *soongGenerator) sharedLibsDir(tgt tgtType) string           { return "" }
func (g *soongGenerator) sourceOutputDir(m *generateCommon) string   { return "" }
func (g *soongGenerator) binaryOutputDir(m *binary) string           { return "" }
func (g *soongGenerator) staticLibOutputDir(m *staticLibrary) string { return "" }
func (g *soongGenerator) sharedLibOutputDir(m *sharedLibrary) string { return "" }
func (g *soongGenerator) kernelModOutputDir(m *kernelModule) string  { return "" }

func (g *soongGenerator) init(*blueprint.Context, *bobConfig) {}

func templateApplierHook(ctx android.LoadHookContext, m blueprint.Module) {
	templateApplier(m, getConfig(ctx), ctx)
}

func featureApplierHook(ctx android.LoadHookContext, m blueprint.Module) {
	featureApplier(m, getConfig(ctx), ctx)
}

// Bob modules that need Soong to run LoadHooks need to implement this
// interface.
type soongLoadHookProvider interface {
	soongLoadHook(android.LoadHookContext, blueprint.Module)
}

func (gs *generateSource) soongLoadHook(ctx android.LoadHookContext, m blueprint.Module) {
	// Flatten features and expand templates
	featureApplierHook(ctx, m)
	templateApplierHook(ctx, m)
}

func soongRegisterModule(name string, mf factoryWithConfig) {

	// Create a closure adapting Bob's module factories to the format Soong uses.
	factory := func() android.Module {
		bpModule, properties := mf(soongGetConfig())
		// This type assertion should always pass as long as every Bob
		// module type embeds moduleBase
		soongModule := bpModule.(android.Module)

		for _, property := range properties {
			soongModule.AddProperties(property)
		}

		if h, ok := bpModule.(soongLoadHookProvider); ok {
			android.AddLoadHook(bpModule, func(ctx android.LoadHookContext) {
				h.soongLoadHook(ctx, bpModule)
			})
		}

		return soongModule
	}
	android.RegisterModuleType(name, factory)
}

func init() {
	registerModuleTypes(soongRegisterModule)
}